-- Line items related queries

-- name: CreateLineItem :one
INSERT INTO line_items (
    bill_id,
    amount_cents,
    currency,
    description,
    incurred_at,
    reference_id,
    metadata,
    idempotency_key
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetLineItem :one
SELECT * FROM line_items WHERE id = $1;

-- name: GetLineItemsByBill :many
SELECT * FROM line_items WHERE bill_id = $1 ORDER BY incurred_at DESC;

-- name: UpdateLineItem :one
UPDATE line_items 
SET amount_cents = $2, description = $3, updated_at = NOW()
WHERE id = $1 
RETURNING *;

-- name: DeleteLineItem :exec
DELETE FROM line_items WHERE id = $1;

-- name: GetTotalAmountByBill :one
SELECT COALESCE(SUM(amount_cents), 0) as total_amount_cents 
FROM line_items 
WHERE bill_id = $1;
