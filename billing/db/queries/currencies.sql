-- Currencies related queries

-- name: GetCurrency :one
SELECT * FROM currencies WHERE code = $1 AND enabled = true;
