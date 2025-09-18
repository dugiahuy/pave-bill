package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// BillingPeriodWorkflowParams contains parameters for starting the billing workflow
type BillingPeriodWorkflowParams struct {
	BillID    int32     `json:"bill_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// BillingPeriodWorkflow manages the lifecycle of a billing period
func BillingPeriod(ctx workflow.Context, params BillingPeriodWorkflowParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting billing period workflow", "billID", params.BillID, "startTime", params.StartTime, "endTime", params.EndTime)

	startTime := params.StartTime
	now := workflow.Now(ctx)
	if startTime.After(now) {
		waitDuration := startTime.Sub(now)
		logger.Info("Waiting for start time", "billID", params.BillID, "waitDuration", waitDuration)
		err := workflow.Sleep(ctx, waitDuration)
		if err != nil {
			return err
		}
		logger.Info("Start time reached, beginning active period", "billID", params.BillID)
	}

	activeDuration := params.EndTime.Sub(params.StartTime)
	if activeDuration <= 0 {
		logger.Warn("End time is before start time, closing immediately", "billID", params.BillID)
		return closeBill(ctx, params.BillID, "invalid_period")
	}

	timer := workflow.NewTimer(ctx, activeDuration)

	addLineItemCh := workflow.GetSignalChannel(ctx, AddLineItemSignalName)
	closeBillCh := workflow.GetSignalChannel(ctx, CloseBillSignalName)

	err := activateBill(ctx, params.BillID)
	if err != nil {
		logger.Error("Failed to activate bill", "billID", params.BillID, "error", err)
		return err
	}

	billClosed := false

	logger.Info("Entering active billing period", "billID", params.BillID, "duration", activeDuration)

	for !billClosed {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(addLineItemCh, func(c workflow.ReceiveChannel, more bool) {
			var signal AddLineItemSignal
			c.Receive(ctx, &signal)
			logger.Info("Tracking line item addition", "billID", params.BillID, "lineItemID", signal.LineItemID)
			err := updateBillTotal(ctx, params.BillID)
			if err != nil {
				logger.Error("Failed to recalculate bill total after line item addition", "billID", params.BillID, "lineItemID", signal.LineItemID, "error", err)
			} else {
				logger.Info("Successfully recalculated bill total after line item addition", "billID", params.BillID, "lineItemID", signal.LineItemID)
			}
		})

		selector.AddReceive(closeBillCh, func(c workflow.ReceiveChannel, more bool) {
			var signal CloseBillSignal
			c.Receive(ctx, &signal)
			logger.Info("Received manual close bill signal", "billID", params.BillID, "reason", signal.Reason)

			err := closeBill(ctx, params.BillID, signal.Reason)
			if err != nil {
				logger.Error("Failed to close bill manually", "error", err)
			} else {
				logger.Info("Successfully closed bill manually", "billID", params.BillID)
				billClosed = true
			}
		})

		selector.AddFuture(timer, func(f workflow.Future) {
			logger.Info("Auto-closing bill due to end time reached", "billID", params.BillID)

			err := closeBill(ctx, params.BillID, "auto_close")
			if err != nil {
				logger.Error("Failed to auto-close bill", "error", err)
			} else {
				logger.Info("Successfully auto-closed bill", "billID", params.BillID)
				billClosed = true
			}
		})

		selector.Select(ctx)
	}

	logger.Info("Billing period workflow completed", "billID", params.BillID)
	return nil
}

// closeBill executes the CloseBill activity
func closeBill(ctx workflow.Context, billID int32, reason string) error {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    15 * time.Second,
			MaximumAttempts:    6,
		},
	}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)
	return workflow.ExecuteActivity(activityCtx, CloseBillActivity, billID, reason).Get(ctx, nil)
}

// activateBill executes the ActivateBill activity
func activateBill(ctx workflow.Context, billID int32) error {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    5,
		},
	}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)
	return workflow.ExecuteActivity(activityCtx, ActivateBillActivity, billID).Get(ctx, nil)
}

// updateBillTotal executes the UpdateBillTotal activity to recalculate totals
func updateBillTotal(ctx workflow.Context, billID int32) error {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    500 * time.Millisecond,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Second,
			MaximumAttempts:    4,
		},
	}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)
	return workflow.ExecuteActivity(activityCtx, UpdateBillTotalActivity, billID).Get(ctx, nil)
}
