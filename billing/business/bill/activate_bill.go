package bill

import (
	"context"
)

// ActivateBill transitions a bill from pending to active status
func (b *business) ActivateBill(ctx context.Context, billID int32) error {
	return b.stateMachine.TransitionToActive(ctx, billID)
}
