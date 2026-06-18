-- name: CreateItem :one
-- Mint a newly listed item into the marketplace registry
INSERT INTO items (name, type, owner_id, base_price, list_price, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetItemByID :one
SELECT * FROM items
WHERE id = $1;

-- name: GetItemByIDForUpdate :one
-- Lock the asset to prevent double-selling or concurrent auction setups on the same item
SELECT * FROM items
WHERE id = $1
FOR UPDATE;

-- name: ListAvailableItems :many
-- Retrieve all unmarketed items up for grabs, sorted by recent listings
SELECT * FROM items
WHERE status = 'available'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAvailableItems :one
SELECT COUNT(*) FROM items
WHERE status = 'available';

-- name: ListAvailableItemsByType :many
SELECT * FROM items
WHERE status = 'available'
  AND type = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAvailableItemsByType :one
SELECT COUNT(*) FROM items
WHERE status = 'available'
  AND type = $1;

-- name: UpdateItemStatus :one
UPDATE items
SET
    status     = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: TransferItemOwnership :one
-- Atomically shift asset ownership and update status when an order completes or auction closes
UPDATE items
SET
    owner_id   = $2,
    status     = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;