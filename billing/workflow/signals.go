package workflow

const (
	// Signal names
	AddLineItemSignalName = "add-line-item"
	CloseBillSignalName   = "close-bill"
)

// AddLineItemSignal contains simplified data for adding a line item to a bill
// The activity will query the database for full line item details and handle bill total updates
type AddLineItemSignal struct {
	LineItemID int32 `json:"line_item_id"`
}

// CloseBillSignal contains data for manually closing a bill
type CloseBillSignal struct {
	Reason   string `json:"reason"`
	ClosedBy string `json:"closed_by"`
}
