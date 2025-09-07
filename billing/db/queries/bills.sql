-- Bills related queries

-- name: CreateBill :one
INSERT INTO bills (
    currency,
    status,
    start_time,
    end_time,
    idempotency_key
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetBill :one
SELECT * FROM bills WHERE id = $1;

-- name: UpdateBillStatus :one
UPDATE bills 
SET status = $2, updated_at = NOW()
WHERE id = $1 
RETURNING *;

-- name: GetBillByIdempotencyKey :one
SELECT * FROM bills WHERE idempotency_key = $1;

-- name: ListBills :many
SELECT * FROM bills 
ORDER BY created_at DESC 
LIMIT $1 OFFSET $2;

-- name: CountBills :one
SELECT COUNT(*) FROM bills;

-- name: UpdateBillTotal :one
UPDATE bills 
SET total_amount_cents = (
    SELECT COALESCE(SUM(amount_cents), 0) 
    FROM line_items 
    WHERE bill_id = $1
), updated_at = NOW()
WHERE bills.id = $2 
RETURNING *;
