package main

import (
	"context"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"log"
	"time"
)

type OperationParameter struct {
	FigiInterval string
	Op           int
	Ind1         IndicatorParameter
	Cl           int
	Ind2         IndicatorParameter
}

type IndicatorParameter struct {
	IndicatorType IndicatorType
	BarType       BarType
	Coef          int
}

func (param OperationParameter) getCandleData() *CandleData {
	return getCandleData(param.FigiInterval)
}

func (param OperationParameter) getFigi() string {
	return param.getCandleData().getFigi()
}

func (indicatorType IndicatorType) getFunction(data *CandleData) funGet {
	switch indicatorType {
	case IndicatorTypeSma:
		return data.getSma
	case IndicatorTypeEma:
		return data.getEma
	case IndicatorTypeDema:
		return data.getDema
	case IndicatorTypeTema:
		return data.getTema
	case IndicatorTypeTemaZero:
		return data.getTemaZero
	}
	return nil
}

func (indicator IndicatorParameter) getValue(data *CandleData, i int) float64 {
	return indicator.IndicatorType.getFunction(data)(indicator.Coef, i, indicator.BarType)
}

//func newCandleEvent(tinkoff *Tinkoff, candle tf.Candle) {
//	data := getCandleData(candle.FIGI, candle.Interval)
//
//	if data.upsertCandle(candle) {
//		for _, parameter := range OperationParameters[candle.FIGI][candle.Interval] {
//			checkOpening(tinkoff, data, candle, parameter)
//		}
//	}
//
//	data.saveToStorage()
//}

func (tinkoff *Tinkoff) Open(figi string, lots int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	placedOrder, err := tinkoff.getApiClient().MarketOrder(ctx, tf.DefaultAccount, figi, lots, tf.BUY)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("%+v\n", placedOrder)
}

func (tinkoff *Tinkoff) Close(figi string, lots int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	placedOrder, err := tinkoff.getApiClient().MarketOrder(ctx, tf.DefaultAccount, figi, lots, tf.SELL)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("%+v\n", placedOrder)
}

func checkOpening(tinkoff *Tinkoff, data *CandleData, candle tf.Candle, parameter OperationParameter) {
	i := data.index() - 1
	val1 := parameter.Ind1.getValue(data, i)
	val2 := parameter.Ind2.getValue(data, i)
	tinkoff.Open(candle.FIGI, 1)
	if val1*10000/val2 >= float64(10000+parameter.Op) {
		tinkoff.Open(candle.FIGI, 1)
	}
}

//func (parameter OperationParameter) getFigiInterval() string {
//	return figiInterval(parameter.Figi, parameter.Interval)
//}
