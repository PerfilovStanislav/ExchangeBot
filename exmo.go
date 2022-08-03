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

func (exmo *Exmo) asyncDownloadHistoryCandles(operations []OperationParameter) {
	parallel(0, len(operations), func(ys <-chan int) {
		for y := range ys {
			exmo.downloadHistoryCandles(operations[y])
		}
	})
}

func (exmo *Exmo) downloadHistoryCandles(operation OperationParameter) {
	candleData := operation.getCandleData()
	candleData.Candles = make(map[BarType][]float64)

	endDate := time.Now().Unix()
	startDate := time.Now().AddDate(0, -2, 0).Unix()

	figi := candleData.getPairName()

	for startDate < endDate {
		from := startDate
		to := startDate + 125*3600

		candleHistory := exmo.apiGetCandles(figi, "60", from, to)

		startDate = to

		if candleHistory.isEmpty() {
			continue
		}
		for _, c := range candleHistory.Candles {
			candleData.upsertCandle(Candle{
				c.L,
				c.O,
				c.C,
				c.H,
				(c.L + c.O) * 0.5,
				(c.L + c.C) * 0.5,
				(c.L + c.H) * 0.5,
				(c.O + c.C) * 0.5,
				(c.O + c.H) * 0.5,
				(c.C + c.H) * 0.5,
				(c.L + c.O + c.C) / 3.0,
				(c.L + c.O + c.H) / 3.0,
				(c.L + c.C + c.H) / 3.0,
				(c.O + c.C + c.H) / 3.0,
				time.Unix(c.T/1000, 0),
			})
		}
		fmt.Printf("Кол-во свечей: %d\n", candleData.len())
	}

	candleData.fillIndicators()
	candleData.save()
}

func (exmo *Exmo) listenCandles(operations []OperationParameter) {
	_, _ = scheduler.Cron("0 * * * *").Do(exmo.checkOperation, operations)
}

func (exmo *Exmo) checkOperation(operations []OperationParameter) {
	exmo.asyncDownloadNewCandle(getUniqueOperations(operations))
	if exmo.isOrderOpened() {
		exmo.checkForClose()
	} else {
		exmo.checkForOpen(operations)
	}
}

func (exmo *Exmo) checkForOpen(operations []OperationParameter) {
	exmo.apiGetUserInfo()

	for _, operation := range operations {
		candleData := operation.getCandleData()

		index := candleData.index()
		v1 := candleData.fillIndicator(index, operation.Ind1)
		v2 := candleData.fillIndicator(index, operation.Ind2)
		candleData.save()

		color.HiBlue("%s\n", time.Now().Format("02.01.06 15:04:05"))
		percentsForOpen := v1 * 10000 / v2 / float64(10000+operation.Op)
		if percentsForOpen >= 1.0 {
			pair := operation.getPairName()
			money := exmo.getCurrencyBalance(getRightCurrency(pair)) * exmo.AvailableDeposit
			buyOrder := exmo.apiBuy(pair, money)
			if buyOrder.isSuccess() {
				candleOpenPrice := exmo.getCurrentCandle(pair).O
				exmo.OpenedOrder = OpenedOrder{
					OperationParameter: operation,
					OpenedPrice:        candleOpenPrice,
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
			fmt.Printf("%.4f operation:%+v v1:%f v2:%f\n", percentsForOpen, operation, v1, v2)
		}
	}
}

func (exmo *Exmo) getCurrentCandle(pair string) ExmoCandle {
	dt := (time.Now().Unix() / 3600) * 3600
	return exmo.apiGetCandles(pair, resolution, dt, dt).Candles[0]
}

func (exmo *Exmo) asyncDownloadNewCandle(operations []OperationParameter) {
	dt := ((time.Now().Unix() / 3600) - 1) * 3600

	parallel(0, len(operations), func(ys <-chan int) {
		for y := range ys {
			exmo.downloadNewCandle(resolution, dt, operations[y])
		}
	})
}

func (exmo *Exmo) downloadNewCandle(resolution string, dt int64, operation OperationParameter) {
	var candleHistory ExmoCandleHistoryResponse
	for i := 1; i <= 10; i++ {
		candleHistory = exmo.apiGetCandles(operation.getPairName(), resolution, dt, dt)
		if !candleHistory.isEmpty() {
			break
		} else if i == 20 {
			return // empty
		} else {
			time.Sleep(time.Millisecond * 50)
		}
	}

	candleData := operation.getCandleData()
	c := candleHistory.Candles[0]
	candleData.upsertCandle(Candle{
		c.L,
		c.O,
		c.C,
		c.H,
		(c.L + c.O) * 0.5,
		(c.L + c.C) * 0.5,
		(c.L + c.H) * 0.5,
		(c.O + c.C) * 0.5,
		(c.O + c.H) * 0.5,
		(c.C + c.H) * 0.5,
		(c.L + c.O + c.C) / 3.0,
		(c.L + c.O + c.H) / 3.0,
		(c.L + c.C + c.H) / 3.0,
		(c.O + c.C + c.H) / 3.0,
		time.Unix(c.T/1000, 0),
	})

	candleData.save()
}

func (exmo *Exmo) checkForClose() {
	openedOrder := exmo.OpenedOrder
	pair := openedOrder.getPairName()
	o := exmo.getCurrentCandle(pair).O
	percentsForClose := o * 10000 / openedOrder.OpenedPrice / float64(10000+openedOrder.Cl)
	fmt.Printf("Percents to close: %f", percentsForClose)
	if percentsForClose >= 1.0 {
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

func (exmo *Exmo) apiGetCurrentPrice() CurrentPriceResponse {
	params := ApiParams{}

	bts, err := exmo.apiQuery("ticker", params)

	var response CurrentPriceResponse
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
