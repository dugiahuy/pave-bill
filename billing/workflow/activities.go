package workflow

import (
	"context"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"encore.app/billing/business/bill"
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

// CloseBillActivity closes a bill and calculates final amounts
func CloseBillActivity(ctx context.Context, billID int32, reason string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing close bill activity", "billID", billID, "reason", reason)

	if activityDeps == nil || activityDeps.BillBusiness == nil {
		logger.Error("Activity dependencies not set")
		return temporal.NewApplicationError("activity dependencies not initialized", "DependencyError")
	}

	err := activityDeps.BillBusiness.CloseBill(ctx, billID, reason)
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

	err := activityDeps.BillBusiness.ActivateBill(ctx, billID)
	if err != nil {
		logger.Error("Failed to activate bill", "billID", billID, "error", err)
		return err
	}

	logger.Info("Successfully activated bill", "billID", billID)
	return nil
}

// UpdateBillTotalActivity recalculates the bill total based on current line items
func UpdateBillTotalActivity(ctx context.Context, billID int32) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing update bill total activity", "billID", billID)

	if activityDeps == nil || activityDeps.BillBusiness == nil {
		logger.Error("Activity dependencies not set")
		return temporal.NewApplicationError("activity dependencies not initialized", "DependencyError")
	}

	err := activityDeps.BillBusiness.UpdateBillTotal(ctx, billID)
	if err != nil {
		logger.Error("Failed to update bill total", "billID", billID, "error", err)
		return temporal.NewNonRetryableApplicationError("failed to update bill total", "BILL_TOTAL_UPDATE_FAILED", err)
	}

	logger.Info("Successfully updated bill total", "billID", billID)
	return nil
}
