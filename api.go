package main

type ApiInterface interface {
	init()
	downloadCandles(candleData *CandleData, operation OperationParameter)
	listenCandles(operations []OperationParameter)
}
