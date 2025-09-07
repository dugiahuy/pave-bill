package bill

import (
	"context"
)

// ActivateBill transitions a bill from pending to active status
func (b *business) ActivateBill(ctx context.Context, billID int32) error {
	// Use the direct state machine transition method instead of ExecuteWithLock
	// to avoid potential deadlocks when called from workflow activities
	return b.stateMachine.TransitionToActive(ctx, billID)
}