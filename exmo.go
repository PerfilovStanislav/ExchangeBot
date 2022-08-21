package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"time"
)

var exmo Exmo

const resolution = "60"

type ApiParams map[string]string

func (exmo *Exmo) init() {
	exmo.Key = os.Getenv("exmo.key")
	exmo.Secret = os.Getenv("exmo.secret")
	exmo.AvailableDeposit = s2f(os.Getenv("available.deposit"))
	exmo.apiGetUserInfo()
}

func (exmo *Exmo) downloadHistoryCandlesForStrategies(strategies []Strategy) {
	for _, strategy := range strategies {
		candleData := strategy.getCandleData()
		candleData.Candles = make(map[BarType][]float64)
		exmo.downloadHistoryCandles(candleData)
	}
}

func (exmo *Exmo) downloadHistoryCandles(candleData *CandleData) {
	endDate := time.Now().Unix()
	startDate := time.Now().AddDate(0, -2, 0).Unix()

	for startDate < endDate {
		from := startDate
		to := startDate + 125*3600

		candleHistory := exmo.apiGetCandles(candleData.Pair, resolution, from, to)

		startDate = to

		if candleHistory.isEmpty() {
			continue
		}
		for _, c := range candleHistory.Candles {
			candleData.upsertCandle(c.transform())
		}
		fmt.Printf("%s - %s +%d\n",
			time.Unix(from, 0).Format("02.01.06 15"),
			time.Unix(to, 0).Format("02.01.06 15"),
			len(candleHistory.Candles),
		)
		time.Sleep(time.Millisecond * time.Duration(50))
	}
	fmt.Printf("Кол-во свечей: %d\n", candleData.len())

	candleData.fillIndicators()
	candleData.save()
}

func (exmo *Exmo) downloadNewCandleForOperations(strategies []Strategy) {
	dt := ((time.Now().Unix() / 3600) - 1) * 3600

	for _, strategy := range strategies {
		exmo.downloadNewCandle(resolution, dt, strategy)
	}
}

func (exmo *Exmo) downloadNewCandle(resolution string, dt int64, strategy Strategy) {
	var candleHistory ExmoCandleHistoryResponse
	for i := 1; i <= 10; i++ {
		candleHistory = exmo.apiGetCandles(strategy.Pair, resolution, dt, dt)
		if !candleHistory.isEmpty() {
			break
		} else if i == 20 {
			return // empty
		} else {
			time.Sleep(time.Millisecond * 50)
		}
	}

	candleData := strategy.getCandleData()
	c := candleHistory.Candles[0]
	candleData.upsertCandle(c.transform())

	candleData.save()
}

func (exmo *Exmo) listenCandles(strategies []Strategy) {
	_, _ = scheduler.Cron("0 * * * *").Do(exmo.checkOperation, strategies)
}

func (exmo *Exmo) checkOperation(strategies []Strategy) {
	exmo.downloadNewCandleForOperations(getUniqueStrategies(strategies))
	if exmo.isOrderOpened() {
		exmo.checkForClose()
	} else {
		exmo.checkForOpen(strategies)
	}
}

func (exmo *Exmo) checkForOpen(strategies []Strategy) {
	exmo.apiGetUserInfo()

	for _, strategy := range strategies {
		candleData := strategy.getCandleData()

		index := candleData.index()
		v1 := candleData.fillIndicator(index, strategy.Ind1)
		v2 := candleData.fillIndicator(index, strategy.Ind2)
		candleData.save()

		color.HiBlue("%s\n", time.Now().Format("02.01.06 15:04:05"))
		percentsForOpen := v1 * 10000 / v2 / float64(10000+strategy.Op)
		if percentsForOpen >= 1.0 {
			pair := strategy.Pair
			money := exmo.getCurrencyBalance(getRightCurrency(pair)) * exmo.AvailableDeposit
			buyOrder := exmo.apiBuy(pair, money)
			if buyOrder.isSuccess() {
				candleOpenPrice := exmo.getCurrentCandle(pair).O
				exmo.OpenedOrder = OpenedOrder{
					Strategy:    strategy,
					OpenedPrice: candleOpenPrice,
				}
				color.HiGreen("SUCCESS order open->")

				// выставляем стоп лосс
				quantity := exmo.getCurrencyBalance(getLeftCurrency(pair))
				StopLossPrice := candleOpenPrice * 0.8
				stopLossOrder := exmo.apiSetStopLoss(pair, quantity, StopLossPrice)
				if stopLossOrder.isSuccess() {
					exmo.StopLossOrderId = stopLossOrder.ParentOrderID
				} else {
					color.HiRed("ERROR set stopLoss %+v", stopLossOrder)
				}
				tgBot.newOrderOpened(pair, candleOpenPrice, quantity, StopLossPrice)
			} else {
				color.HiRed("ERROR order open->")
			}
			fmt.Printf("OpenedOrder:%+v\nOrder:%+v\n\n", exmo.OpenedOrder, buyOrder)
		} else {
			fmt.Printf("%.4f strategy:%+v v1:%f v2:%f\n", percentsForOpen, strategy, v1, v2)
		}
	}
}

func (exmo *Exmo) getCurrentCandle(pair string) ExmoCandle {
	dt := (time.Now().Unix() / 3600) * 3600
	return exmo.apiGetCandles(pair, resolution, dt, dt).Candles[0]
}

func (exmo *Exmo) checkForClose() {
	openedOrder := exmo.OpenedOrder
	pair := openedOrder.Pair
	o := exmo.getCurrentCandle(pair).O
	percentsToClose := o * 10000 / openedOrder.OpenedPrice / float64(10000+openedOrder.Cl)
	fmt.Printf("Percents to close: %f", percentsToClose)
	if percentsToClose >= 1.0 {
		exmo.apiGetUserInfo()
		quantity := exmo.getCurrencyBalance(getLeftCurrency(pair))
		order := exmo.apiClose(pair, quantity)

		if order.isSuccess() {
			exmo.apiCancelStopLoss(exmo.StopLossOrderId)
			exmo.OpenedOrder = OpenedOrder{}
			exmo.StopLossOrderId = 0
			color.HiGreen("SUCCESS order close->")
			tgBot.orderClosed(pair, o, quantity)
		} else {
			color.HiRed("ERROR order close->")
		}
		fmt.Printf("Operation:%+v\nOrder:%+v\n\n", openedOrder, order)
	}
}

func (exmo *Exmo) apiGetCandles(symbol, resolution string, from, to int64) ExmoCandleHistoryResponse {
	params := ApiParams{
		"symbol":     symbol,
		"resolution": resolution,
		"from":       i2s(from),
		"to":         i2s(to),
	}

	bts, err := exmo.apiQuery("candles_history", params)

	var candleHistory ExmoCandleHistoryResponse
	err = json.Unmarshal(bts, &candleHistory)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	return candleHistory
}

func (exmo *Exmo) apiBuy(pair string, money float64) OrderResponse {
	params := ApiParams{
		"pair":     pair,
		"quantity": f2s(money),
		"price":    "0",
		"type":     "market_buy_total",
	}

	return exmo.apiCreateOrder(params)
}

func (exmo *Exmo) apiSetStopLoss(pair string, coins float64, price float64) StopOrderResponse {
	const decimals = 100000000

	params := ApiParams{
		"pair":          pair,
		"quantity":      f2s(coins),
		"trigger_price": f2s(math.Round(price*decimals) / decimals),
		"type":          "sell",
	}
	bts, err := exmo.apiQuery("stop_market_order_create", params)

	var response StopOrderResponse
	err = json.Unmarshal(bts, &response)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	return response
}

func (exmo *Exmo) apiCancelStopLoss(parentOrderId int64) {
	params := ApiParams{
		"parent_order_id": i2s(parentOrderId),
	}
	_, err := exmo.apiQuery("stop_market_order_cancel", params)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}
}

func (exmo *Exmo) apiCreateOrder(params ApiParams) OrderResponse {
	bts, err := exmo.apiQuery("order_create", params)

	var response OrderResponse
	err = json.Unmarshal(bts, &response)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	return response
}

func (exmo *Exmo) apiClose(symbol string, quantity float64) OrderResponse {
	params := ApiParams{
		"pair":     symbol,
		"quantity": f2s(quantity),
		"price":    "0",
		"type":     "market_sell",
	}

	bts, err := exmo.apiQuery("order_create", params)

	var response OrderResponse
	err = json.Unmarshal(bts, &response)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	return response
}

func (exmo *Exmo) apiGetUserInfo() UserInfoResponse {
	params := ApiParams{}

	bts, err := exmo.apiQuery("user_info", params)

	var response UserInfoResponse
	err = json.Unmarshal(bts, &response)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	exmo.Balance = response.Balances

	return response
}

func (exmo *Exmo) getCurrencyBalance(symbol Currency) float64 {
	r := reflect.ValueOf(exmo.Balance)
	f := reflect.Indirect(r).FieldByName(string(symbol))
	return f.Float()
}

func (exmo *Exmo) apiQuery(method string, params ApiParams) ([]byte, error) {
	postParams := url.Values{}
	postParams.Add("nonce", nonce())
	if params != nil {
		for k, value := range params {
			postParams.Add(k, value)
		}
	}
	postContent := postParams.Encode()

	sign := doSign(postContent, exmo.Secret)

	req, _ := http.NewRequest("POST", "https://api.exmo.com/v1.1/"+method, bytes.NewBuffer([]byte(postContent)))
	req.Header.Set("Key", exmo.Key)
	req.Header.Set("Sign", sign)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(postContent)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		return nil, errors.New("http status: " + resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

func (exmo *Exmo) isOrderOpened() bool {
	return exmo.OpenedOrder.isEmpty() == false
}

func nonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func doSign(message string, secret string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(message))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
