package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/gob"
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

func (exmo *Exmo) init() *Exmo {
	exmo.Key = os.Getenv("exmo.key")
	exmo.Secret = os.Getenv("exmo.secret")
	exmo.AvailableDeposit = s2f(os.Getenv("available.deposit"))
	exmo.apiGetUserInfo()
	exmo.restore()

	return exmo
}

func (exmo *Exmo) showBalance() {
	color.HiYellow("Balance %+v", exmo.Balance)
}

func (exmo *Exmo) restore() bool {
	fileName := exmo.getFileName()
	if !fileExists(fileName) {
		return false
	}
	dataIn := ReadFromFile(fileName)
	dec := gob.NewDecoder(bytes.NewReader(dataIn))
	_ = dec.Decode(&(exmo.OpenedOrder))

	return true
}

func (exmo *Exmo) backup() {
	dataOut := EncodeToBytes(exmo.OpenedOrder)
	_ = os.WriteFile(exmo.getFileName(), dataOut, 0644)
}

func (exmo *Exmo) getFileName() string {
	return fmt.Sprintf("exmo.dat")
}

func (exmo *Exmo) downloadHistoryCandlesForStrategies(strategies []Strategy) {
	for _, strategy := range strategies {
		candleData := strategy.getCandleData()
		candleData.Candles = make(map[BarType][]float64)
		exmo.downloadPairCandles(candleData)
	}
}

func (exmo *Exmo) downloadPairCandles(candleData *CandleData) {
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
	}
	fmt.Printf("Кол-во свечей: %d\n", candleData.len())

	candleData.fillIndicators()
	candleData.save()
}

func (exmo *Exmo) downloadNewCandleForStrategies(strategies []Strategy) {
	for _, strategy := range strategies {
		candle := exmo.downloadNewCandle(-1, strategy.Pair)

		if !candle.isEmpty() {
			candleData := strategy.getCandleData()
			candleData.upsertCandle(candle)
			candleData.save()
		}
	}
}

func (exmo *Exmo) downloadNewCandle(index int64, pair string) Candle {
	dt := ((time.Now().Unix() / 3600) + index) * 3600

	var candleHistory ExmoCandleHistoryResponse
	for i := 1; i <= 50; i++ {
		candleHistory = exmo.apiGetCandles(pair, resolution, dt, dt)
		if !candleHistory.isEmpty() {
			return candleHistory.Candles[0].transform()
		}
	}

	return Candle{}
}

func (exmo *Exmo) listenCandles(strategies []Strategy) {
	_, _ = scheduler.Cron("0 * * * *").Do(exmo.checkOperation, strategies)
}

func (exmo *Exmo) checkOperation(strategies []Strategy) {
	color.HiBlue("%s\n", time.Now().Format("02.01.06 15:04:05"))
	exmo.downloadNewCandleForStrategies(getUniqueStrategies(strategies))
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

		percentsForOpen := 10000 * v1 / v2 / float64(10000+strategy.Op)
		if percentsForOpen > 1.0 {
			pair := strategy.Pair
			candle := exmo.downloadNewCandle(0, pair)
			if candle.isEmpty() {
				continue
			}
			money := exmo.getCurrencyBalance(getRightCurrency(pair)) * exmo.AvailableDeposit
			buyOrder := exmo.apiBuy(pair, money)
			if buyOrder.isSuccess() {
				exmo.OpenedOrder = OpenedOrder{
					Strategy:    strategy,
					OpenedPrice: candle.O,
				}
				color.HiGreen("SUCCESS order open->")

				// выставляем стоп лосс
				exmo.apiGetUserInfo()
				quantity := exmo.getCurrencyBalance(getLeftCurrency(pair))
				stopLossPrice := candle.O * 0.8
				stopLossOrder := exmo.apiSetStopLoss(pair, quantity, stopLossPrice)
				if stopLossOrder.isSuccess() {
					exmo.OpenedOrder.StopLossOrderId = stopLossOrder.ParentOrderID
				} else {
					color.HiRed("ERROR set stopLoss %+v", stopLossOrder)
				}
				exmo.backup()

				takeProfit := candle.O * float64(10000+strategy.Tp) / 10000
				screen := candleData.drawBars(takeProfit, stopLossPrice)
				tgBot.newOrderOpened(pair, candle.O, stopLossPrice, screen)
			} else {
				color.HiRed("ERROR order open->")
			}
			fmt.Printf("OpenedOrder:%+v\nOrder:%+v\n\n", exmo.OpenedOrder, buyOrder)
		} else {
			fmt.Printf("%.4f strategy:%+v v1:%f v2:%f\n", percentsForOpen, strategy, v1, v2)
		}
	}
}

func (exmo *Exmo) checkForClose() {
	openedOrder := exmo.OpenedOrder
	pair := openedOrder.Pair
	candle := exmo.downloadNewCandle(0, pair)
	if candle.isEmpty() {
		time.Sleep(time.Second * 30)
		candle = exmo.downloadNewCandle(0, pair)
		if candle.isEmpty() {
			return
		}
	}
	percentsToClose := candle.O * 10000 / openedOrder.OpenedPrice / float64(10000+openedOrder.Tp)
	fmt.Printf("Percents to close: %f\n", percentsToClose)
	if percentsToClose >= 1.0 {
		exmo.apiCancelStopLoss(exmo.OpenedOrder.StopLossOrderId)
		exmo.OpenedOrder.StopLossOrderId = 0

		exmo.apiGetUserInfo()
		quantity := exmo.getCurrencyBalance(getLeftCurrency(pair))
		order := exmo.apiClose(pair, quantity)

		if order.isSuccess() {
			exmo.OpenedOrder = OpenedOrder{}
			exmo.backup()
			color.HiGreen("SUCCESS order close->")
			tgBot.orderClosed(pair, candle.O)
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
	time.Sleep(time.Millisecond * 50)

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
