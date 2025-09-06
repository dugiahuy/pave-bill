package model

import (
	"time"
)

type Bill struct {
	ID               int32
	Currency         string
	Status           BillStatus
	CloseReason      *string
	ErrorMessage     *string
	TotalAmountCents int64
	StartTime        time.Time
	EndTime          time.Time
	BilledAt         *time.Time
	IdempotencyKey   string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type BillStatus string

const (
	BillStatusPending BillStatus = "pending"
	BillStatusActive  BillStatus = "active"
	BillStatusClosing BillStatus = "closing"
	BillStatusClosed  BillStatus = "closed"
	BillStatusFailed  BillStatus = "failed"
)
