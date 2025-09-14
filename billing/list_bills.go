package billing

import (
	"context"

	"encore.dev/rlog"

	"encore.app/billing/model"
)

type GetBillsRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

type GetBillsResponse struct {
	Bills      []model.Bill `json:"bills"`
	TotalCount int64        `json:"total_count"`
	Limit      int          `json:"limit"`
	Offset     int          `json:"offset"`
}

// encore:api public path=/v1/bills method=GET
func (s *Service) ListBills(ctx context.Context, req *GetBillsRequest) (*GetBillsResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	bills, totalCount, err := s.business.ListBills(ctx, int32(req.Limit), int32(req.Offset))
	if err != nil {
		rlog.Error("failed to get bills", "error", err)
		return nil, err
	}

	response := &GetBillsResponse{
		Bills:      make([]model.Bill, len(bills)),
		TotalCount: totalCount,
		Limit:      req.Limit,
		Offset:     req.Offset,
	}

	for i, bill := range bills {
		response.Bills[i] = *bill
	}

	return response, nil
}
