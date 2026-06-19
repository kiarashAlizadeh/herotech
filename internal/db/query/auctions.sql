-- name: CreateAuction :one
INSERT INTO auctions (item_id, seller_id, start_price, ends_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAuction :one
SELECT * FROM auctions WHERE id = $1;

-- name: GetAuctionForUpdate :one
-- Lock the active auction state to safely evaluate incoming bids sequentially
SELECT * FROM auctions WHERE id = $1 FOR UPDATE;

-- name: ListActiveAuctions :many
-- Fetch all live auctions with embedded core item metadata for the storefront
SELECT a.*, i.name as item_name, i.type as item_type
FROM auctions a
JOIN items i ON a.item_id = i.id
WHERE a.status = 'active'
ORDER BY a.ends_at ASC
LIMIT $1 OFFSET $2;

-- name: CountActiveAuctions :one
SELECT COUNT(*) FROM auctions
WHERE status = 'active';

-- name: CountActiveAuctionsBySeller :one
-- Guard rail query to verify a guild hasn't bypassed the limit of 5 concurrent active auctions
SELECT COUNT(*) FROM auctions
WHERE seller_id = $1 AND status = 'active';

-- name: UpdateAuctionBid :one
-- Update the current leading bid, record the frontrunner, and apply potential time extensions
UPDATE auctions
SET highest_bid = $2,
    winner_id = $3,
    ends_at = $4
WHERE id = $1
RETURNING *;

-- name: CloseAuction :one
-- Terminate the auction lifecycle, finalizing the status and sealing the winner
UPDATE auctions
SET status = $2,
    winner_id = $3
WHERE id = $1
RETURNING *;