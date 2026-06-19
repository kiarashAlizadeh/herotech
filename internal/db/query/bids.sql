-- name: CreateBid :one
INSERT INTO bids (auction_id, bidder_id, amount, is_active)
VALUES ($1, $2, $3, TRUE)
RETURNING *;

-- name: GetActiveBidByBidder :one
SELECT * FROM bids 
WHERE auction_id = $1 AND bidder_id = $2 AND is_active = TRUE;

-- name: GetBidForUpdate :one
-- Lock a specific bid allocation during direct cancellations
SELECT * FROM bids WHERE id = $1 FOR UPDATE;

-- name: DeactivateBid :one
-- Flag a bid as inactive when it is outbid or manually retracted by the user
UPDATE bids
SET is_active = FALSE
WHERE id = $1
RETURNING *;

-- name: DeactivateAuctionBids :many
-- Bulk invalidate all remaining active bids once an auction reaches its final conclusion
UPDATE bids
SET is_active = FALSE
WHERE auction_id = $1
RETURNING *;