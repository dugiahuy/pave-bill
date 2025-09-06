package service

import (
	"encore.app/billing/service/bill"
	"encore.app/billing/store"
)

// Services holds all business services
type Services struct {
	Bill bill.Service
}

// NewServices creates a new services container
func NewServices(repo *store.Store) Services {
	billService := bill.NewBillService(repo.Bills)

	return Services{
		Bill: billService,
	}
}
