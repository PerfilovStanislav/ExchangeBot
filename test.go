package main

import (
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	//_ "github.com/lib/pq"
)

func (candleData *CandleData) getIndicatorValue(indicator IndicatorParameter) []float64 {
	return candleData.Indicators[indicator.IndicatorType][indicator.Coef][indicator.BarType]
}

func (candleData *CandleData) getIndicatorRatio(operation OperationParameter, index int) float64 {
	return candleData.getIndicatorValue(operation.Ind1)[index] / candleData.getIndicatorValue(operation.Ind2)[index]
}
