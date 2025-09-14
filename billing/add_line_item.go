package billing

import (
	"context"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"

	"encore.app/billing/model"
	"encore.app/billing/workflow"
)

type CreateLineItemRequest struct {
	IdempotencyKey string `header:"X-Idempotency-Key" json:"-"`

	Currency    string `json:"currency" validate:"required,len=3,alpha"`
	AmountCents int64  `json:"amount_cents" validate:"required,min=1"`
	Description string `json:"description" validate:"required,max=255"`
	ReferenceID string `json:"reference_id" validate:"required,max=100"`
}

type LineItemResponse struct {
	LineItem model.LineItem `json:"line_item"`
}

// encore:api public path=/v1/bills/:id/line_items method=POST tag:idempotency
func (s *Service) AddLineItem(ctx context.Context, id int32, req *CreateLineItemRequest) (*LineItemResponse, error) {
	if id <= 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid bill ID"}
	}

	lineItem := &model.LineItem{
		BillID:         id,
		Currency:       req.Currency,
		AmountCents:    req.AmountCents,
		Description:    req.Description,
		ReferenceID:    req.ReferenceID,
		IncurredAt:     time.Now(),
		IdempotencyKey: req.IdempotencyKey,
	}

	result, err := s.business.AddLineItemToBill(ctx, id, lineItem)
	if err != nil {
		rlog.Error("failed to create line item", "error", err, "bill_id", id)
		return nil, err
	}

	// Signal workflow asynchronously - don't block the response
	go func() {
		signalCtx := context.Background()
		err := s.signalAddLineItem(signalCtx, result.BillWorkflowID, result.ID)
		if err != nil {
			rlog.Error("failed to signal workflow", "error", err, "workflow_id", result.BillWorkflowID, "line_item_id", result.ID)
		}
	}()

	return &LineItemResponse{
		LineItem: *result,
	}, nil
}

// Validate implements validation for CreateLineItemRequest
func (r *CreateLineItemRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: err.Error()}
	}

	return nil
}

// signalAddLineItem sends a signal to the workflow to process the new line item
func (s *Service) signalAddLineItem(ctx context.Context, workflowID string, lineItemID int32) error {
	signal := workflow.AddLineItemSignal{
		LineItemID: lineItemID,
	}

	return s.temporal.SignalWorkflow(ctx, workflowID, "", workflow.AddLineItemSignalName, signal)
}
