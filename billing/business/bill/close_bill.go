package bill

import (
	"context"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
)

// Close handles closing a bill with proper locking, state transitions, and error handling
func (b *business) CloseBill(ctx context.Context, id int32, reason string) error {
	return b.stateMachine.GetBillWithLock(ctx, id, func(currentBill bills.Bill) error {
		switch currentBill.Status {
		case string(model.BillStatusClosed):
			// Bill is already closed - idempotent operation
			return nil

		case string(model.BillStatusPending):
			return b.stateMachine.TransitionToClosedTx(ctx, id, reason)

		case string(model.BillStatusActive):
			// Full closing process for active bills
			// Step 1: Set to closing state
			err := b.stateMachine.TransitionToClosingTx(ctx, id, reason)
			if err != nil {
				return err
			}

			// Step 2: Recalculate final bill total within the same transaction
			// In reality, this would also involve finalizing many other aspects
			err = b.stateMachine.UpdateBillTotalTx(ctx, id)
			if err != nil {
				// Set error status
				errorMsg := "failed to calculate final bill total: " + err.Error()
				failureErr := b.stateMachine.TransitionToFailureStateTx(ctx, id, errorMsg)
				if failureErr != nil {
					return failureErr // Return failure transition error if it fails
				}
				return &errs.Error{Code: errs.Internal, Message: errorMsg} // Return original error
			}

			// Step 3: Set final status to closed
			return b.stateMachine.TransitionToClosedTx(ctx, id, reason)

		default:
			return &errs.Error{Code: errs.InvalidArgument, Message: "invalid bill status for closure"}
		}
	})
}
