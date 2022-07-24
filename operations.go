package main

import (
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
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

func (param OperationParameter) getPairName() string {
	return param.getCandleData().getPairName()
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
