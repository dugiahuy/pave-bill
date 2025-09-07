package billing

import (
	"context"

	"encore.dev/beta/errs"
	"encore.dev/rlog"

	"encore.app/billing/model"
)

type CloseBillRequest struct {
	Reason string `json:"reason" validate:"required,max=255"`
}

type CloseBillResponse struct {
	Bill model.Bill `json:"bill"`
}

// encore:api public path=/v1/bills/:id/close method=POST tag:idempotency
func (s *Service) CloseBill(ctx context.Context, id int32, req *CloseBillRequest) (*CloseBillResponse, error) {
	if id <= 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid bill ID"}
	}

	// Close the bill using the domain service, which returns the full bill with line items
	err := s.business.CloseBill(ctx, id, req.Reason)
	if err != nil {
		rlog.Error("failed to close bill", "error", err, "id", id)
		return nil, err
	}

	bill, err := s.business.GetBill(ctx, id)
	if err != nil {
		rlog.Error("failed to get closed bill", "error", err, "id", id)
		return nil, err
	}

	return &CloseBillResponse{
		Bill: *bill,
	}, nil
}

// Validate implements validation for CloseBillRequest
func (r *CloseBillRequest) Validate() error {
	if err := validate.Struct(r); err != nil {
		return &errs.Error{Code: errs.InvalidArgument, Message: err.Error()}
	}

	return nil
}
