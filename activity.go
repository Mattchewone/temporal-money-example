package app

import (
	"context"
	"fmt"
)

func Withdraw(ctx context.Context, transferDetails TransferDetails, account Account) (Account, error) {
	fmt.Printf(
		"\nWithdrawing $%f from account %s. ReferenceId: %s\n",
		transferDetails.Amount,
		transferDetails.FromAccount,
		transferDetails.ReferenceID,
	)
	account.Withdraw(transferDetails)
	return account, nil
}

func Deposit(ctx context.Context, transferDetails TransferDetails, account Account) (Account, error) {
	fmt.Printf(
		"\nDepositing $%f into account %s. ReferenceId: %s\n",
		transferDetails.Amount,
		transferDetails.ToAccount,
		transferDetails.ReferenceID,
	)
	account.Deposit(transferDetails)
	return account, nil
}
