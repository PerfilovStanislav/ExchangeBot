package main

import (
	"reflect"
)

type ExmoCandleHistoryResponse struct {
	S       string       `json:"s"`
	Candles []ExmoCandle `json:"candles"`
}

type ExmoCandle struct {
	T int64   `json:"t"`
	O float64 `json:"o"`
	C float64 `json:"c"`
	H float64 `json:"h"`
	L float64 `json:"l"`
}

type OrderResponse struct {
	Result   bool   `json:"result"`
	Error    string `json:"error"`
	OrderID  int    `json:"order_id"`
	ClientID int    `json:"client_id"`
}

func (response OrderResponse) isSuccess() bool {
	return response.Error == ""
}

type StopOrderResponse struct {
	ClientID         int    `json:"client_id"`
	ParentOrderID    int64  `json:"parent_order_id"`
	ParentOrderIDStr string `json:"parent_order_id_str"`
}

func (response StopOrderResponse) isSuccess() bool {
	return response.ParentOrderID > 0
}

type UserInfoResponse struct {
	UID        int                     `json:"uid"`
	ServerDate int                     `json:"server_date"`
	Balances   CurrencyBalanceResponse `json:"balances"`
	//Reserved   CurrencyBalanceResponse `json:"reserved"`
}

type CurrentPriceResponse struct {
	ALGO_USDT CurrentPrice `json:"ALGO_USDT"`
}

type Price float64

type CurrentPrice struct {
	BuyPrice Price `json:"buy_price,string"`
}

func (response CurrentPriceResponse) getPrice(pair string) Price {
	r := reflect.ValueOf(response)
	f := reflect.Indirect(r).FieldByName(pair)
	x := f.Interface().(CurrentPrice)

	return x.BuyPrice
}
