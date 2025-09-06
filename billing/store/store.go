package store

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"encore.app/billing/store/bills"
	"encore.app/billing/store/lineitems"
)

// Store combines all domain-specific repositories
type Store struct {
	Bills     bills.Querier
	LineItems lineitems.Querier
}

// NewStore creates a new Store with all domain queriers
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		Bills:     bills.New(db),
		LineItems: lineitems.New(db),
	}
}

// WithTx creates a new Store using a transaction
func (r *Store) WithTx(tx interface{}) *Store {
	// Note: You'll need to implement transaction handling for each domain
	// This is a placeholder for transaction support
	return &Store{
		Bills:     r.Bills,
		LineItems: r.LineItems,
	}
}
