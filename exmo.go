package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"time"
)

var exmo Exmo

type ApiParams map[string]string

func (exmo *Exmo) init() {
	exmo.Key = os.Getenv("exmo.key")
	exmo.Secret = os.Getenv("exmo.secret")
	exmo.apiGetUserInfo()
}

func (exmo *Exmo) downloadCandles(candleData *CandleData, param OperationParameter) {
	candleData.Candles = make(map[BarType][]float64)

	endDate := time.Now().Unix()
	startDate := time.Now().AddDate(0, -2, 0).Unix()

	figi, _ := getFigiAndInterval(candleData.FigiInterval)

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
				c.O, c.C, c.H, c.L, time.Unix(c.T/1000, 0),
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
	if exmo.isOrderOpened() {
		exmo.checkForClose()
	} else {
		exmo.checkForOpen(operations)
	}
}

func (exmo *Exmo) checkForOpen(operations []OperationParameter) {
	exmo.apiGetUserInfo()

	dt := ((time.Now().Unix() / 3600) - 1) * 3600
	resolution := "60"

_operations:
	for _, operation := range operations {
		var candleHistory ExmoCandleHistoryResponse
		for i := 1; i <= 10; i++ {
			candleHistory = exmo.apiGetCandles(operation.getPairName(), resolution, dt, dt)
			if candleHistory.isEmpty() {
				fmt.Printf("EMPTY i:%2d dt:%d %+v \n", i, dt, operation)
				if i == 10 {
					continue _operations
				}
				time.Sleep(time.Millisecond * 250)
			} else {
				break
			}
		}

		candleData := operation.getCandleData()
		c := candleHistory.Candles[0]
		candleData.upsertCandle(Candle{
			c.O, c.C, c.H, c.L, time.Unix(c.T/1000, 0),
		})

		index := candleData.index()
		v1 := candleData.fillIndicator(index, operation.Ind1)
		v2 := candleData.fillIndicator(index, operation.Ind2)
		candleData.save()

		if v1*10000/v2 >= float64(10000+operation.Op) {
			pair := operation.getPairName()
			order := exmo.apiBuy(pair, exmo.calculateQuantity(pair))
			if order.Error != "" {
				exmo.OpenedOrder = operation
				fmt.Println("SUCCESS order open->")
			} else {
				fmt.Println("ERROR order open->")
			}
			fmt.Printf("Operation:%+v\nOrder:%+v\n\n", operation, order)
		} else {
			fmt.Println(time.Now().Format(time.RFC850))
		}
	}
}

func (exmo *Exmo) calculateQuantity(pair string) float64 {
	_, rightCurrency := getCurrencies(pair)
	money := exmo.getCurrencyBalance(rightCurrency)

	return money / float64(exmo.apiGetCurrentPrice().getPrice(pair)) * 0.98
}

func (exmo *Exmo) checkForClose() {
	operation := exmo.OpenedOrder
	exmo.apiGetUserInfo()
	pair := operation.getPairName()
	leftCurrency, _ := getCurrencies(pair)
	quantity := exmo.getCurrencyBalance(leftCurrency)
	order := exmo.apiClose(pair, quantity)

	if order.Error != "" {
		exmo.OpenedOrder = operation
		fmt.Println("SUCCESS order close->")
	} else {
		fmt.Println("ERROR order close->")
	}
	fmt.Printf("Operation:%+v\nOrder:%+v\n\n", operation, order)
}

func (exmo *Exmo) apiGetCandles(symbol, resolution string, from, to int64) ExmoCandleHistoryResponse {
	params := ApiParams{
		"symbol":     symbol,
		"resolution": resolution,
		"from":       strconv.FormatInt(from, 10),
		"to":         strconv.FormatInt(to, 10),
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

func (exmo *Exmo) apiBuy(symbol string, quantity float64) OrderResponse {
	params := ApiParams{
		"pair":     symbol,
		"quantity": f2s(quantity),
		"price":    "0",
		"type":     "market_buy",
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

	exmo.CurrencyBalance = response.Balances

	return response
}

func (exmo *Exmo) getCurrencyBalance(symbol Currency) float64 {
	r := reflect.ValueOf(exmo.CurrencyBalance)
	f := reflect.Indirect(r).FieldByName(string(symbol))
	return f.Float()
}

func (exmo *Exmo) getPrevCurrencyBalance(symbol Currency) float64 {
	r := reflect.ValueOf(exmo.PrevCurrencyBalance)
	f := reflect.Indirect(r).FieldByName(string(symbol))
	return f.Float()
}

func (exmo *Exmo) getCurrentPrice(symbol Currency) float64 {
	r := reflect.ValueOf(exmo.PrevCurrencyBalance)
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
