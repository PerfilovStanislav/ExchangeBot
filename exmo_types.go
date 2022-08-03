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
	USDT Currency = "USDT"
	ETC  Currency = "ETC"
)

type CurrencyBalanceResponse struct {
	USDT Price `json:"USDT,string"`
	ETC  Price `json:"ETC,string"`
}
