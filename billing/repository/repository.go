package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/currencies"
	"encore.app/billing/repository/lineitems"
)

// Repository combines all domain-specific repositories
type Repository struct {
	Bills      bills.Querier
	LineItems  lineitems.Querier
	Currencies currencies.Querier
}

// NewRepository creates a new Repository with all domain queriers
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		Bills:      bills.New(db),
		LineItems:  lineitems.New(db),
		Currencies: currencies.New(db),
	}
}

// WithTx creates a new Repository using a transaction
func (r *Repository) WithTx(tx interface{}) *Repository {
	// Note: You'll need to implement transaction handling for each domain
	// This is a placeholder for transaction support
	return &Repository{
		Bills:      r.Bills,
		LineItems:  r.LineItems,
		Currencies: r.Currencies,
	}
}
