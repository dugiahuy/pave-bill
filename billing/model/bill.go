package model

import (
	"time"
)

type Bill struct {
	ID               int32      `json:"id"`
	Currency         string     `json:"currency"`
	Status           BillStatus `json:"status"`
	CloseReason      *string    `json:"close_reason,omitempty"`
	ErrorMessage     *string    `json:"error_message,omitempty"`
	TotalAmountCents int64      `json:"total_amount_cents"`
	StartTime        time.Time  `json:"start_time"`
	EndTime          time.Time  `json:"end_time"`
	BilledAt         *time.Time `json:"billed_at,omitempty"`
	IdempotencyKey   string     `json:"idempotency_key"`
	WorkflowID       *string    `json:"workflow_id,omitempty"`
	LineItems        []LineItem `json:"line_items,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type BillStatus string

const (
	BillStatusPending           BillStatus = "pending"
	BillStatusActive            BillStatus = "active"
	BillStatusClosing           BillStatus = "closing"
	BillStatusClosed            BillStatus = "closed"
	BillStatusAttentionRequired BillStatus = "attention_required"
)
