package workflow

import (
	"context"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"encore.app/billing/business/bill"
	"encore.app/billing/model"
)

// ActivityDependencies holds the dependencies needed by activities
// Now using the unified business layer
type ActivityDependencies struct {
	BillBusiness bill.Business
}

var activityDeps *ActivityDependencies

// SetActivityDependencies sets the dependencies for activities
func SetActivityDependencies(billBusiness bill.Business) {
	activityDeps = &ActivityDependencies{
		BillBusiness: billBusiness,
	}
}

// AddLineItemActivity processes a line item addition and updates bill totals
func AddLineItemActivity(ctx context.Context, billID int32, signal AddLineItemSignal) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing add line item activity", "billID", billID, "lineItemID", signal.LineItemID)

	// The line item was already created in the API, but now we need to update the bill total
	// This ensures the workflow maintains consistency and proper sequencing of operations

	// Update bill total to include the new line item
	err := activityDeps.BillBusiness.UpdateBillTotal(ctx, billID)
	if err != nil {
		logger.Error("Failed to update bill total after adding line item",
			"billID", billID,
			"lineItemID", signal.LineItemID,
			"error", err)

		// This is a critical error - bill total is out of sync
		return temporal.NewNonRetryableApplicationError("failed to update bill total", "BILL_TOTAL_UPDATE_FAILED", err)
	}

	logger.Info("Successfully processed add line item activity",
		"billID", billID,
		"lineItemID", signal.LineItemID)

	return nil
}

// CloseBillActivity closes a bill and calculates final amounts
func CloseBillActivity(ctx context.Context, billID int32, reason string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing close bill activity", "billID", billID, "reason", reason)

	if activityDeps == nil || activityDeps.BillBusiness == nil {
		logger.Error("Activity dependencies not set")
		return temporal.NewApplicationError("activity dependencies not initialized", "DependencyError")
	}

	// Get the current bill to ensure it exists and get current totals
	currentBill, err := activityDeps.BillBusiness.GetBill(ctx, billID)
	if err != nil {
		logger.Error("Failed to get bill", "billID", billID, "error", err)
		return err
	}

	// Check if bill is already closed
	if currentBill.Status == model.BillStatusClosed {
		logger.Info("Bill is already closed", "billID", billID)
		return nil // Not an error, just already closed
	}

	// Update bill status to closed and recalculate totals one final time
	// The UpdateBillTotal will recalculate the total from all line items
	err = activityDeps.BillBusiness.CloseBill(ctx, billID, reason)
	if err != nil {
		logger.Error("Failed to close bill", "billID", billID, "error", err)
		return err
	}

	logger.Info("Successfully closed bill", "billID", billID, "reason", reason)
	return nil
}

// ActivateBillActivity transitions a bill to active status when the billing period begins
func ActivateBillActivity(ctx context.Context, billID int32) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing activate bill activity", "billID", billID)

	if activityDeps == nil || activityDeps.BillBusiness == nil {
		logger.Error("Activity dependencies not set")
		return temporal.NewApplicationError("activity dependencies not initialized", "DependencyError")
	}

	// Activate the bill
	err := activityDeps.BillBusiness.ActivateBill(ctx, billID)
	if err != nil {
		logger.Error("Failed to activate bill", "billID", billID, "error", err)
		return err
	}

	logger.Info("Successfully activated bill", "billID", billID)
	return nil
}
