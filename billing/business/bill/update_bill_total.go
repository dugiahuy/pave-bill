package bill

import (
	"context"

	"encore.app/billing/repository/bills"
)

// UpdateBillTotal recalculates and updates the total amount for a bill based on its line items
// Uses row-level locking to prevent race conditions when multiple line items are added concurrently
func (b *business) UpdateBillTotal(ctx context.Context, billID int32) error {
	return b.stateMachine.GetBillWithLock(ctx, billID, func(currentBill bills.Bill) error {
		// The actual total calculation happens in the database using UpdateBillTotal
		// This ensures the calculation is atomic and uses the latest line items
		return b.stateMachine.UpdateBillTotalTx(ctx, billID)
	})
}
