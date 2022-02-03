package app

const TransferMoneyTaskQueue = "TRANSFER_MONEY_TASK_QUEUE"
const SignalChannelName = "TRANSFER_SIGNAL"

type TransferDetails struct {
	Amount      float32
	FromAccount string
	ToAccount   string
	ReferenceID string
}

var RouteTypes = struct {
	DEPOSIT  string
	WITHDRAW string
}{
	DEPOSIT:  "deposit",
	WITHDRAW: "withdraw",
}

type RouteSignal struct {
	Route string
}

type MoneySignal struct {
	Route           string
	TransferDetails TransferDetails
}
