package billing

import (
	"context"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"

	"encore.app/billing/model"
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
	result, err := s.services.Bill.Create(ctx, &model.Bill{
		Currency:       req.Currency,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		rlog.Error("failed to create bill", "error", err)
		return nil, err
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
	} else {
		if r.EndTime.Before(r.StartTime) {
			return &errs.Error{Code: errs.InvalidArgument, Message: "end_time must be after start_time"}
		}
	}

	return nil
}

// ==================================================================

// Encore comes with a built-in local development dashboard for
// exploring your API, viewing documentation, debugging with
// distributed tracing, and more:
//
//     http://localhost:9400
//

// ==================================================================

// Next steps
//
// 1. Deploy your application to the cloud
//
//     git add -A .
//     git commit -m 'Commit message'
//     git push encore
//
// 2. To continue exploring Encore, check out some of these topics:
//
// 	  Defining Services:			 https://encore.dev/docs/go/primitives/services
// 	  Defining APIs:				 https://encore.dev/docs/go/primitives/defining-apis
//    Using SQL databases:  		 https://encore.dev/docs/go/primitives/databases
//    Using Pub/Sub:  				 https://encore.dev/docs/go/primitives/pubsub
//    Authenticating users: 		 https://encore.dev/docs/go/develop/auth
//    Building a REST API:  		 https://encore.dev/docs/go/tutorials/rest-api
//	  Building an Event-Driven app:  https://encore.dev/docs/go/tutorials/uptime
//    Building a Slack bot: 		 https://encore.dev/docs/go/tutorials/slack-bot
//	  Example apps repo:			 https://github.com/encoredev/examples
