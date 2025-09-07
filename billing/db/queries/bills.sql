-- Bills related queries

-- name: CreateBill :one
INSERT INTO bills (
    currency,
    status,
    start_time,
    end_time,
    idempotency_key,
    workflow_id
) VALUES (
    $1, $2, $3, $4, $5, $6
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

-- name: GetBillForUpdate :one
SELECT * FROM bills WHERE id = $1 FOR UPDATE;

-- name: UpdateBillTotal :one
UPDATE bills 
SET total_amount_cents = (
    SELECT COALESCE(SUM(amount_cents), 0) 
    FROM line_items 
    WHERE bill_id = $1
), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateBillClosure :one
UPDATE bills 
SET status = $2, 
    close_reason = $3,
    error_message = $4,
    updated_at = NOW()
WHERE id = $1 
RETURNING *;
