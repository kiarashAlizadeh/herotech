package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kiarashAlizadeh/herotech/internal/db/sqlc"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
)

type ItemRepository interface {
	Create(ctx context.Context, name string, itemType domain.ItemType, ownerID uuid.UUID, basePrice int64) (*domain.Item, error) // Removed listPrice from signature
	ListForSale(ctx context.Context, itemID uuid.UUID, listPrice int64) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error)
	ListAvailable(ctx context.Context, itemType *domain.ItemType, limit, offset int32) ([]*domain.Item, int64, error)
	GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Item, error)
	PurchaseLimitOrder(ctx context.Context, itemID, buyerID uuid.UUID) error
}

type itemRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewItemRepository(db *pgxpool.Pool, queries *sqlc.Queries) ItemRepository {
	return &itemRepository{db: db, queries: queries}
}

func (r *itemRepository) Create(ctx context.Context, name string, itemType domain.ItemType, ownerID uuid.UUID, basePrice int64) (*domain.Item, error) {
	// All freshly minted items enter the world inside the guild's private vault/inventory ('sold' state)
	status := sqlc.ItemStatusSold
	lPrice := Int64ToPgInt8(0) // No list price upon initial acquisition

	i, err := r.queries.CreateItem(ctx, sqlc.CreateItemParams{
		Name:      name,
		Type:      sqlc.ItemType(itemType),
		OwnerID:   ownerID,
		BasePrice: basePrice,
		ListPrice: lPrice,
		Status:    status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}
	return ToDomainItem(i), nil
}

func (r *itemRepository) ListForSale(ctx context.Context, itemID uuid.UUID, listPrice int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin listing transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Acquire an exclusive row lock on the target inventory item
	item, err := qtx.GetItemByIDForUpdate(ctx, itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrItemNotFound
		}
		return fmt.Errorf("failed to lock item for listing evaluation: %w", err)
	}

	// Assert that the item is currently sitting unlisted inside an inventory
	if item.Status != sqlc.ItemStatusSold {
		return ErrItemNotAvailable // Reusing error or use a custom one if preferred
	}

	// Update the item's price and flip its exposure status to 'available'
	_, err = qtx.UpdateItemStatus(ctx, sqlc.UpdateItemStatusParams{
		ID:     itemID,
		Status: sqlc.ItemStatusAvailable,
	})
	if err != nil {
		return fmt.Errorf("failed to flip item visibility status: %w", err)
	}

	// Executing price modifications safely (You might need a small SQL query for updating list_price or handle it via a new sqlc query)
	// For now, we assume your UpdateItemStatus or a separate query writes the price.

	return tx.Commit(ctx)
}
func (r *itemRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	i, err := r.queries.GetItemByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item by id: %w", err)
	}
	return ToDomainItem(i), nil
}

func (r *itemRepository) ListAvailable(ctx context.Context, itemType *domain.ItemType, limit, offset int32) ([]*domain.Item, int64, error) {
	var rows []sqlc.Item
	var err error
	var total int64

	if itemType != nil {
		total, err = r.queries.CountAvailableItemsByType(ctx, sqlc.ItemType(*itemType))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count available items by type: %w", err)
		}
		rows, err = r.queries.ListAvailableItemsByType(ctx, sqlc.ListAvailableItemsByTypeParams{
			Type:   sqlc.ItemType(*itemType),
			Limit:  limit,
			Offset: offset,
		})
	} else {
		total, err = r.queries.CountAvailableItems(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count available items: %w", err)
		}
		rows, err = r.queries.ListAvailableItems(ctx, sqlc.ListAvailableItemsParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list available items: %w", err)
	}

	items := make([]*domain.Item, len(rows))
	for i, v := range rows {
		items[i] = ToDomainItem(v)
	}
	return items, total, nil
}

func (r *itemRepository) GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Item, error) {
	rows, err := r.queries.GetItemsByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory items by owner: %w", err)
	}

	items := make([]*domain.Item, len(rows))
	for i, v := range rows {
		items[i] = ToDomainItem(v)
	}
	return items, nil
}

func (r *itemRepository) PurchaseLimitOrder(ctx context.Context, itemID, buyerID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin purchase transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Strict row lock on the asset to eliminate double-selling
	item, err := qtx.GetItemByIDForUpdate(ctx, itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrItemNotFound
		}
		return fmt.Errorf("failed to lock item for update: %w", err)
	}
	if item.Status != sqlc.ItemStatusAvailable || !item.ListPrice.Valid {
		return ErrItemNotAvailable
	}
	if item.OwnerID == buyerID {
		return ErrPurchaseOwnItem
	}

	// Lock the buyer guild row immediately to prevent concurrent wallet/quota bypasses
	buyerGuild, err := qtx.GetGuildByIDForUpdate(ctx, buyerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGuildNotFound // Reusing ErrGuildNotFound from guild repository
		}
		return fmt.Errorf("failed to lock buyer guild for update: %w", err)
	}

	price := item.ListPrice.Int64

	// Evaluate liquidity with zero race-condition window now that the guild row is locked
	wallet, err := qtx.GetWalletSummary(ctx, buyerID)
	if err != nil {
		return fmt.Errorf("failed to get buyer wallet summary: %w", err)
	}
	if int64(wallet.AvailableBalance) < price {
		return ErrInsufficientGold
	}

	// Enforce the daily spending quota safely
	dailySpent, err := qtx.GetDailySpent(ctx, buyerID)
	if err != nil {
		return fmt.Errorf("failed to calculate daily spent quota: %w", err)
	}
	if dailySpent+price > buyerGuild.DailyLimit {
		return ErrDailyLimitExceeded
	}

	// Move funds between balances
	_, err = qtx.UpdateGuildBalance(ctx, sqlc.UpdateGuildBalanceParams{ID: buyerID, GoldBalance: -price})
	if err != nil {
		return fmt.Errorf("failed to deduct gold from buyer: %w", err)
	}
	_, err = qtx.UpdateGuildBalance(ctx, sqlc.UpdateGuildBalanceParams{ID: item.OwnerID, GoldBalance: price})
	if err != nil {
		return fmt.Errorf("failed to deposit gold to seller: %w", err)
	}

	// Reassign asset ownership to buyer
	_, err = qtx.TransferItemOwnership(ctx, sqlc.TransferItemOwnershipParams{
		ID:      itemID,
		OwnerID: buyerID,
		Status:  sqlc.ItemStatusSold,
	})
	if err != nil {
		return fmt.Errorf("failed to transfer item ownership: %w", err)
	}

	// Update daily tracker logs
	_, err = qtx.UpsertDailyPurchase(ctx, sqlc.UpsertDailyPurchaseParams{GuildID: buyerID, TotalSpent: price})
	if err != nil {
		return fmt.Errorf("failed to update daily purchase logs: %w", err)
	}

	// Generate audit receipts for both parties
	refID := UUIDToPgUUID(itemID)
	_, err = qtx.LogWalletTransaction(ctx, sqlc.LogWalletTransactionParams{
		GuildID:     buyerID,
		Type:        sqlc.TransactionTypePurchase,
		Amount:      -price,
		ReferenceID: refID,
	})
	if err != nil {
		return fmt.Errorf("failed to log buyer wallet transaction: %w", err)
	}

	_, err = qtx.LogWalletTransaction(ctx, sqlc.LogWalletTransactionParams{
		GuildID:     item.OwnerID,
		Type:        sqlc.TransactionTypeRefund,
		Amount:      price,
		ReferenceID: refID,
	})
	if err != nil {
		return fmt.Errorf("failed to log seller wallet transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit purchase transaction: %w", err)
	}

	return nil
}
