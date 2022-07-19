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
	"strconv"
	"time"
)

type Exmo struct {
	Key    string
	Secret string
}

var exmo Exmo

type ExmoCandleHistory struct {
	S       string `json:"s"`
	Candles []struct {
		T int64   `json:"t"`
		O float64 `json:"o"`
		C float64 `json:"c"`
		H float64 `json:"h"`
		L float64 `json:"l"`
	} `json:"candles"`
}

type OrderResponse struct {
	Result   bool   `json:"result"`
	Error    string `json:"error"`
	OrderID  int    `json:"order_id"`
	ClientID int    `json:"client_id"`
}

func (exmo *Exmo) init() {
	exmo.Key = os.Getenv("exmo.key")
	exmo.Secret = os.Getenv("exmo.secret")
}

func (exmo *Exmo) downloadCandles(candleData *CandleData, param OperationParameter) {
	candleData.Candles = make(map[BarType][]float64)

	endDate := time.Now().Unix()
	startDate := time.Now().AddDate(0, -2, 0).Unix()

	figi, _ := getFigiAndInterval(candleData.FigiInterval)

	for startDate < endDate {
		from := startDate
		to := startDate + 125*3600

		candleHistory := exmo.getCandles(figi, "60", from, to)

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
	_, _ = scheduler.Cron("0 * * * *").Do(exmo.listenCandleParams, operations)
}

func (candleHistory ExmoCandleHistory) isEmpty() bool {
	return candleHistory.S != ""
}
func (exmo *Exmo) listenCandleParams(operations []OperationParameter) {
	dt := ((time.Now().Unix() / 60) - 1) * 60

	resolution := "60"

_operations:
	for _, operation := range operations {
		var candleHistory ExmoCandleHistory
		for i := 1; i <= 10; i++ {
			candleHistory = exmo.getCandles(operation.getFigi(), resolution, dt, dt)
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

		for _, c := range candleHistory.Candles {
			candleData.upsertCandle(Candle{
				c.O, c.C, c.H, c.L, time.Unix(c.T/1000, 0),
			})
		}

		index := candleData.index()
		v1 := candleData.fillIndicator(index, operation.Ind1)
		v2 := candleData.fillIndicator(index, operation.Ind2)
		candleData.save()

		if v1*10000/v2 >= float64(10000+operation.Op) {
			fmt.Printf("Open order! %+v \n", operation)
		} else {
			fmt.Println("no")
		}
	}
}

func (exmo *Exmo) getCandles(symbol, resolution string, from, to int64) ExmoCandleHistory {
	params := ApiParams{
		"symbol":     symbol,
		"resolution": resolution,
		"from":       strconv.FormatInt(from, 10),
		"to":         strconv.FormatInt(to, 10),
	}

	bts, err := exmo.apiQuery("candles_history", params)

	var candleHistory ExmoCandleHistory
	err = json.Unmarshal(bts, &candleHistory)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	return candleHistory
}

func f2s(x float64) string {
	return fmt.Sprintf("%v", x)
}

func (exmo *Exmo) buy(symbol string, quantity float64) OrderResponse {
	params := ApiParams{
		"pair":     symbol,
		"quantity": f2s(quantity),
		"price":    "0",
		"type":     "market_buy",
	}

	bts, err := exmo.apiQuery("order_create", params)

	var orderResponse OrderResponse
	err = json.Unmarshal(bts, &orderResponse)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}

	return orderResponse
}

type ApiParams map[string]string

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

func nonce() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func doSign(message string, secret string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(message))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
