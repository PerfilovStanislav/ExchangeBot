package main

type Exmo struct {
	Key                 string
	Secret              string
	CurrencyBalance     CurrencyBalanceResponse
	PrevCurrencyBalance CurrencyBalanceResponse
	OpenedOrder         OperationParameter
}

func (candleHistory ExmoCandleHistoryResponse) isEmpty() bool {
	return candleHistory.S != ""
}

func (param OperationParameter) isEmpty() bool {
	return param.FigiInterval == ""
}

type Currency string

const (
	ALGO Currency = "ALGO"
	EXM  Currency = "EXM"
	USD  Currency = "USD"
	USDT Currency = "USDT"
	RUB  Currency = "RUB"
)
