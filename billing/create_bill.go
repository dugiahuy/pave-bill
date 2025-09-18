package billing

import (
	"context"
	"fmt"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"encore.app/billing/model"
	"encore.app/billing/workflow"
)

type CreateBillRequest struct {
	IdempotencyKey string `header:"X-Idempotency-Key" json:"-"`

	Currency  string    `json:"currency" validate:"required,len=3,alpha"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time" validate:"required"`
}

type BillResponse struct {
	Bill model.Bill `json:"bill"`
}

//encore:api public path=/v1/bills method=POST tag:idempotency
func (s *Service) CreateBill(ctx context.Context, req *CreateBillRequest) (*BillResponse, error) {
	if req.StartTime.IsZero() {
		req.StartTime = time.Now()
	}
	result, err := s.business.CreateBill(ctx, &model.Bill{
		Currency:       req.Currency,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		rlog.Error("failed to create bill", "error", err)
		return nil, err
	}

	// Start Temporal workflow for bill lifecycle management
	if wfErr := s.startBillingWorkflow(ctx, result); wfErr != nil {
		// We intentionally do not fail the overall request, but we emit structured context
		rlog.Error("workflow start issue", "bill_id", result.ID, "workflow_id", fmt.Sprintf("bill-%s", result.IdempotencyKey), "error", wfErr)
	}

	return &BillResponse{
		Bill: *result,
	}, nil
}

// Validate implements validation for CreateBillRequest using go-playground/validator
func (r *CreateBillRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: err.Error()}
	}

	if !r.StartTime.IsZero() {
		if r.StartTime.Before(time.Now()) {
			return &errs.Error{Code: errs.InvalidArgument, Message: "start_time must be in the future"}
		}

		if r.EndTime.Before(time.Now()) {
			return &errs.Error{Code: errs.InvalidArgument, Message: "end_time must be in the future"}
		}

		if r.EndTime.Before(r.StartTime) {
			return &errs.Error{Code: errs.InvalidArgument, Message: "end_time must be after start_time"}
		}
	} else {
		// StartTime is zero (will be set to now in the API)
		if r.EndTime.Before(time.Now()) {
			return &errs.Error{Code: errs.InvalidArgument, Message: "end_time must be after start_time"}
		}
	}

	return nil
}

// startBillingWorkflow starts a Temporal workflow for bill lifecycle management
func (s *Service) startBillingWorkflow(ctx context.Context, bill *model.Bill) error {
	workflowID := fmt.Sprintf("bill-%s", bill.IdempotencyKey)

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: taskQueue,
	}

	params := workflow.BillingPeriodWorkflowParams{
		BillID:    bill.ID,
		StartTime: bill.StartTime,
		EndTime:   bill.EndTime,
	}

	_, err := s.temporal.ExecuteWorkflow(ctx, options, workflow.BillingPeriod, params)
	if err != nil {
		// Distinguish AlreadyStarted (benign) vs real failure
		if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
			rlog.Info("workflow already started", "bill_id", bill.ID, "workflow_id", workflowID)
			return nil
		}
		return fmt.Errorf("execute workflow %s: %w", workflowID, err)
	}
	return nil
}
