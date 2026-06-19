-- name: LogWalletTransaction :one
-- Append an immutable audit trail entry for every shift in guild finances
INSERT INTO wallet_transactions (guild_id, type, amount, reference_id, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListWalletTransactions :many
-- Extract the full financial audit ledger for a specific guild, reverse chronological order
SELECT * FROM wallet_transactions
WHERE guild_id = $1
ORDER BY created_at DESC;