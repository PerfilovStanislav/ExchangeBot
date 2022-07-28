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
		//"pair":   "ALGO_USDT",
		"limit":  "100",
		"offset": "0",
	}

	bts, err := exmo.apiQuery("user_cancelled_orders", params)

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
