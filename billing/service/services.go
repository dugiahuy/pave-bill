package service

import (
	"encore.app/billing/repository"
	"encore.app/billing/service/bill"
)

// Services holds all business services
type Services struct {
	Bill bill.Service
}

// NewServices creates a new services container
func NewServices(repo *repository.Repository) Services {
	billService := bill.NewBillService(repo.Bills)

	return Services{
		Bill: billService,
	}
}
