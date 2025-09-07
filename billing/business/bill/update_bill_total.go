package bill

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// UpdateBillTotal recalculates and updates the total amount for a bill based on its line items
// No locking needed - just refreshing the total amount
func (b *business) UpdateBillTotal(ctx context.Context, billID int32) error {
	_, err := b.billRepo.UpdateBillTotal(ctx, pgtype.Int4{Int32: billID, Valid: true})
	return err
}
