package main

import (
	"fmt"
)

type TestResponse struct {
	ETC_USD struct {
		Ask [][]string `json:"ask"`
		Bid [][]string `json:"bid"`
	} `json:"ETC_USD"`
}

func (exmo *Exmo) getStopLossOrderId() {
	params := ApiParams{}

	fmt.Println(exmo.apiQuery("user_open_orders", params))
}

func (exmo *Exmo) cancelStopLossOrder() {
	params := ApiParams{
		"parent_order_id": "507081767134440159",
	}

	fmt.Println(exmo.apiQuery("stop_market_order_cancel", params))
}

func calc(rows [][]string, i int64) float64 {
	sum := 0.0
	for _, row := range rows {
		sum += s2f(row[i])
	}
	return sum
}
