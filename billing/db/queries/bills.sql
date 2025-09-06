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
