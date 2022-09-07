package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type TestResponse struct {
	ETC_USD struct {
		Ask [][]string `json:"ask"`
		Bid [][]string `json:"bid"`
	} `json:"ETC_USD"`
}

func (exmo *Exmo) test() {
	params := ApiParams{
		//"date": "1662336000",
		//"limit":  "100",
		//"offset": "0",
		"order_id": "30746429423",
	}

	bts, err := exmo.apiQuery("order_trades", params)

	var response TestResponse
	err = json.Unmarshal(bts, &response)

	bidPrice := calc(response.ETC_USD.Bid, 2)
	bidAmount := calc(response.ETC_USD.Bid, 1)

	askPrice := calc(response.ETC_USD.Ask, 2)
	askAmount := calc(response.ETC_USD.Ask, 1)

	bidAvg := bidPrice / bidAmount
	askAvg := askPrice / askAmount

	avg := askAvg / bidAvg

	fmt.Println(bidPrice, bidAmount, askPrice, askAmount, bidAvg, askAvg, avg)

	if err != nil {
		fmt.Sprintln(err)
		log.Fatalln(err)
	}
	fmt.Println(response)
}

func calc(rows [][]string, i int64) float64 {
	sum := 0.0
	for _, row := range rows {
		sum += s2f(row[i])
	}
	return sum
}
