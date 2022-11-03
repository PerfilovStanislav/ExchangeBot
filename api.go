package main

type ApiInterface interface {
	showBalance()
	downloadHistoryCandlesForStrategies(strategies []Strategy)
	downloadPairCandles(candleData *CandleData)
	listenCandles(strategies []Strategy)
}
