-- Currencies related queries

-- name: GetCurrency :one
SELECT * FROM currencies WHERE code = $1 AND enabled = true;

-- name: GetAllEnabledCurrencies :many
SELECT * FROM currencies WHERE enabled = true ORDER BY code;

-- name: CreateCurrency :one
INSERT INTO currencies (
    code,
    symbol,
    rate,
    enabled
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdateCurrencyRate :one
UPDATE currencies 
SET rate = $2, updated_at = NOW()
WHERE code = $1 
RETURNING *;