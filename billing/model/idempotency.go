package model

import (
	"encoding/json"
	"time"
)

// IdempotencyKey represents the cache key structure
type IdempotencyKey struct {
	Resource string
	Key      string
}

// IdempotencyCacheEntry represents what we store in the cache
type IdempotencyCacheEntry struct {
	Status          string          `json:"status"`
	RequestBodyHash string          `json:"request_body_hash"`
	Response        json.RawMessage `json:"response,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
