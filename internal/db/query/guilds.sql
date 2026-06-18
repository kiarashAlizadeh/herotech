-- name: CreateGuild :one
-- Register a new guild into the market with a baseline configuration
INSERT INTO guilds (name, daily_limit)
VALUES ($1, $2)
RETURNING *;

-- name: ListGuilds :many
SELECT * FROM guilds 
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountGuilds :one
SELECT COUNT(*) FROM guilds;

-- name: GetGuildByID :one
SELECT * FROM guilds
WHERE id = $1;

-- name: GetGuildByIDForUpdate :one
-- Lock the guild row to safely mutate balance and prevent race conditions during checkout or bidding
SELECT * FROM guilds
WHERE id = $1
FOR UPDATE;

-- name: UpdateGuildBalance :one
-- Adjust the guild's gold balance. The WHERE clause acts as a safety guard against negative balances
UPDATE guilds
SET
    gold_balance = gold_balance + $2,
    updated_at   = NOW()
WHERE id = $1
  AND gold_balance + $2 >= 0
RETURNING *;

-- name: GetWalletSummary :one
-- Calculate a complete financial snapshot for a guild, isolating active bid reservations
SELECT
    g.gold_balance                                          AS total_balance,
    COALESCE(SUM(b.amount) FILTER (WHERE b.is_active), 0)::BIGINT  AS reserved_amount,
    g.gold_balance - COALESCE(SUM(b.amount) FILTER (WHERE b.is_active), 0)::BIGINT AS available_balance
FROM guilds g
LEFT JOIN bids b ON b.bidder_id = g.id
WHERE g.id = $1
GROUP BY g.id, g.gold_balance;

-- name: GetDailySpent :one
-- Fetch today's total accumulated spending to enforce the daily anti-monopoly quota
SELECT COALESCE(total_spent, 0)::BIGINT
FROM daily_purchases
WHERE guild_id = $1 AND date = CURRENT_DATE;

-- name: UpsertDailyPurchase :one
-- Log or increment the guild's daily expenditures inside the current calendar date
INSERT INTO daily_purchases (guild_id, date, total_spent)
VALUES ($1, CURRENT_DATE, $2)
ON CONFLICT (guild_id, date) 
DO UPDATE SET total_spent = daily_purchases.total_spent + EXCLUDED.total_spent
RETURNING *;