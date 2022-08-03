package main

type Exmo struct {
	Key              string
	Secret           string
	AvailableDeposit float64
	Balance          CurrencyBalanceResponse
	OpenedOrder      OpenedOrder
	StopLossOrderId  int64
}

type OpenedOrder struct {
	OperationParameter
	OpenedPrice float64
}

func (candleHistory ExmoCandleHistoryResponse) isEmpty() bool {
	return candleHistory.S != ""
}

func (order OpenedOrder) isEmpty() bool {
	return order.OperationParameter.FigiInterval == ""
}

type Currency string

const (
	ALGO Currency = "ALGO"
	EXM  Currency = "EXM"
	USD  Currency = "USD"
	USDT Currency = "USDT"
	RUB  Currency = "RUB"
	ETC  Currency = "ETC"
)
