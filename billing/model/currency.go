package model

type Currency string

const (
	USD Currency = "USD"
	GEL Currency = "GEL"
)

type CurrencyInfo struct {
	ID      int32   `json:"id"`
	Code    string  `json:"code"`
	Symbol  *string `json:"symbol,omitempty"`
	Rate    float64 `json:"rate"`
	Enabled bool    `json:"enabled"`
}

type ConversionResult struct {
	ConvertedAmount int64             `json:"converted_amount"`
	Metadata        *CurrencyMetadata `json:"metadata,omitempty"`
}
