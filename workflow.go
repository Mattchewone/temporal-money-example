package app

import (
	"time"

	"github.com/mitchellh/mapstructure"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type Account struct {
	Amount    float32
	Transfers []TransferDetails
}

func TransferMoney(ctx workflow.Context, account Account) error {
	logger := workflow.GetLogger(ctx)
	// RetryPolicy specifies how to automatically handle retries if an Activity fails.
	retrypolicy := &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    500,
	}
	options := workflow.ActivityOptions{
		// Timeout options specify when to automatically timeout Actvitivy functions.
		StartToCloseTimeout: time.Minute,
		// Optionally provide a customized RetryPolicy.
		// Temporal retries failures by default, this is just an example.
		RetryPolicy: retrypolicy,
	}

	err := workflow.SetQueryHandler(ctx, "getAccount", func(input []byte) (Account, error) {
		return account, nil
	})
	if err != nil {
		logger.Info("SetQueryHandler failed.", "Error", err)
		return err
	}

	channel := workflow.GetSignalChannel(ctx, SignalChannelName)

	for {
		selector := workflow.NewSelector(ctx)
		selector.AddReceive(channel, func(c workflow.ReceiveChannel, _ bool) {
			var signal interface{}
			c.Receive(ctx, &signal)

			var routeSignal RouteSignal
			err := mapstructure.Decode(signal, &routeSignal)
			if err != nil {
				logger.Error("Invalid signal type %v", err)
				return
			}

			switch {
			case routeSignal.Route == RouteTypes.DEPOSIT:
				var message MoneySignal
				err := mapstructure.Decode(signal, &message)
				if err != nil {
					logger.Error("Invalid signal type %v", err)
					return
				}

				ctx = workflow.WithActivityOptions(ctx, options)
				err = workflow.ExecuteActivity(ctx, Deposit, message.TransferDetails, account).Get(ctx, &account)
				if err != nil {
					logger.Error("Invalid ExecuteActivity %v", err)
					return
				}
			case routeSignal.Route == RouteTypes.WITHDRAW:
				var message MoneySignal
				err := mapstructure.Decode(signal, &message)
				if err != nil {
					logger.Error("Invalid signal type %v", err)
					return
				}

				ctx = workflow.WithActivityOptions(ctx, options)
				err = workflow.ExecuteActivity(ctx, Withdraw, message.TransferDetails, account).Get(ctx, &account)
				if err != nil {
					logger.Error("Invalid ExecuteActivity %v", err)
					return
				}
			}
		})

		selector.Select(ctx)

		if account.Amount == 500 {
			break
		}
	}

	return nil
}

func (state *Account) Deposit(item TransferDetails) {
	state.Transfers = append(state.Transfers, item)
	state.Amount += item.Amount
}

func (state *Account) Withdraw(item TransferDetails) {
	state.Transfers = append(state.Transfers, item)
	state.Amount -= item.Amount
}
