package billing

import (
	"context"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"

	"encore.app/billing/model"
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
func (s *Service) CreateLineItem(ctx context.Context, id int, req *CreateLineItemRequest) (*LineItemResponse, error) {
	if id <= 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid bill ID"}
	}

	lineItem := &model.LineItem{
		BillID:         int32(id),
		Currency:       req.Currency,
		AmountCents:    req.AmountCents,
		Description:    req.Description,
		ReferenceID:    req.ReferenceID,
		IncurredAt:     time.Now(),
		IdempotencyKey: req.IdempotencyKey,
	}

	result, err := s.services.LineItem.Create(ctx, lineItem)
	if err != nil {
		rlog.Error("failed to create line item", "error", err, "bill_id", id)
		return nil, err
	}

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
