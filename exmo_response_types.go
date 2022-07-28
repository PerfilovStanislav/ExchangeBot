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

type CurrencyBalanceResponse struct {
	ALGO Price `json:"ALGO,string"`
	CRON Price `json:"CRON,string"`
	EXM  Price `json:"EXM,string"`
	IQN  Price `json:"IQN,string"`
	USD  Price `json:"USD,string"`
	USDT Price `json:"USDT,string"`
	QTUM Price `json:"QTUM,string"`
	RUB  Price `json:"RUB,string"`
	//EUR    Price `json:"EUR,string"`
	//GBP    Price `json:"GBP,string"`
	//PLN    Price `json:"PLN,string"`
	//UAH    Price `json:"UAH,string"`
	//KZT    Price `json:"KZT,string"`
	//BTC    Price `json:"BTC,string"`
	//LTC    Price `json:"LTC,string"`
	//DOGE   Price `json:"DOGE,string"`
	//DASH   Price `json:"DASH,string"`
	//ETH    Price `json:"ETH,string"`
	//WAVES  Price `json:"WAVES,string"`
	//ZEC    Price `json:"ZEC,string"`
	//XRP    Price `json:"XRP,string"`
	//ETC    Price `json:"ETC,string"`
	//BCH    Price `json:"BCH,string"`
	//BTG    Price `json:"BTG,string"`
	//EOS    Price `json:"EOS,string"`
	//XLM    Price `json:"XLM,string"`
	//OMG    Price `json:"OMG,string"`
	//TRX    Price `json:"TRX,string"`
	//ADA    Price `json:"ADA,string"`
	//NEO    Price `json:"NEO,string"`
	//GAS    Price `json:"GAS,string"`
	//ZRX    Price `json:"ZRX,string"`
	//GUSD   Price `json:"GUSD,string"`
	//XEM    Price `json:"XEM,string"`
	//SMART  Price `json:"SMART,string"`
	//QTUM   Price `json:"QTUM,string"`
	//DAI    Price `json:"DAI,string"`
	//MKR    Price `json:"MKR,string"`
	//MNC    Price `json:"MNC,string"`
	//USDC   Price `json:"USDC,string"`
	//ROOBEE Price `json:"ROOBEE,string"`
	//DCR    Price `json:"DCR,string"`
	//XTZ    Price `json:"XTZ,string"`
	//VLX    Price `json:"VLX,string"`
	//ONT    Price `json:"ONT,string"`
	//ONG    Price `json:"ONG,string"`
	//ATOM   Price `json:"ATOM,string"`
	//WXT    Price `json:"WXT,string"`
	//CHZ    Price `json:"CHZ,string"`
	//ONE    Price `json:"ONE,string"`
	//PRQ    Price `json:"PRQ,string"`
	//HAI    Price `json:"HAI,string"`
	//LINK   Price `json:"LINK,string"`
	//UNI    Price `json:"UNI,string"`
	//YFI    Price `json:"YFI,string"`
	//GNY    Price `json:"GNY,string"`
	//XYM    Price `json:"XYM,string"`
	//BTCV   Price `json:"BTCV,string"`
	//DOT    Price `json:"DOT,string"`
	//TON    Price `json:"TON,string"`
	//SGB    Price `json:"SGB,string"`
	//SHIB   Price `json:"SHIB,string"`
	//GMT    Price `json:"GMT,string"`
	//SOL    Price `json:"SOL,string"`
	//EXFI   Price `json:"EXFI,string"`
	//SOLO   Price `json:"SOLO,string"`
	//NEAR   Price `json:"NEAR,string"`
}

type UserInfoResponse struct {
	UID        int                     `json:"uid"`
	ServerDate int                     `json:"server_date"`
	Balances   CurrencyBalanceResponse `json:"balances"`
	Reserved   CurrencyBalanceResponse `json:"reserved"`
}

type CurrentPriceResponse struct {
	ALGO_USDT CurrentPrice `json:"ALGO_USDT"`
	CRON_USDT CurrentPrice `json:"CRON_USDT"`
	IQN_USDT  CurrentPrice `json:"IQN_USDT"`
	QTUM_USD  CurrentPrice `json:"QTUM_USD"`
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
