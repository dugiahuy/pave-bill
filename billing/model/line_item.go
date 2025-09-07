package model

import (
	"time"
)

type LineItem struct {
	ID             int32             `json:"id"`
	BillID         int32             `json:"bill_id"`
	AmountCents    int64             `json:"amount_cents"`
	Currency       string            `json:"currency"`
	Description    string            `json:"description"`
	IncurredAt     time.Time         `json:"incurred_at"`
	ReferenceID    string            `json:"reference_id"`
	Metadata       *CurrencyMetadata `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`

	BillWorkflowID string `json:"-"`
}

func (li *LineItem) SetBillWorkflowID(id string) {
	li.BillWorkflowID = id
}

type CurrencyMetadata struct {
	OriginalAmountCents int64   `json:"original_amount_cents"`
	OriginalCurrency    string  `json:"original_currency"`
	ExchangeRate        float64 `json:"exchange_rate"`
}
