package billing

import (
	"context"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
)

// encore:api public path=/v1/bills/:id method=GET
func (s *Service) GetBill(ctx context.Context, id int) (*BillResponse, error) {
	if id <= 0 {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "invalid bill ID"}
	}

	result, err := s.business.GetBill(ctx, int32(id))
	if err != nil {
		rlog.Error("failed to get bill", "error", err, "id", id)
		return nil, err
	}

	return &BillResponse{
		Bill: *result,
	}, nil
}
