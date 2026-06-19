package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kiarashAlizadeh/herotech/internal/db/sqlc"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
)

type AuctionRepository interface {
	Create(ctx context.Context, itemID, sellerID uuid.UUID, startPrice int64, duration time.Duration) (*domain.Auction, error)
	ListActive(ctx context.Context, limit, offset int32) ([]*domain.Auction, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Auction, error)
	PlaceBidTransaction(ctx context.Context, auctionID, bidderID uuid.UUID, amount int64, extendedEndTime time.Time) error
	CancelBidTransaction(ctx context.Context, auctionID, bidderID uuid.UUID) error
	FinalizeExpiredAuction(ctx context.Context, auctionID uuid.UUID) error
}

type auctionRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewAuctionRepository(db *pgxpool.Pool, queries *sqlc.Queries) AuctionRepository {
	return &auctionRepository{db: db, queries: queries}
}

func (r *auctionRepository) Create(ctx context.Context, itemID, sellerID uuid.UUID, startPrice int64, duration time.Duration) (*domain.Auction, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Assert business restriction: max 5 active auctions allowed per guild
	activeCount, err := qtx.CountActiveAuctionsBySeller(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active auctions: %w", err)
	}
	if activeCount >= 5 {
		return nil, ErrMaxActiveAuctions
	}

	endsAt := time.Now().Add(duration)
	a, err := qtx.CreateAuction(ctx, sqlc.CreateAuctionParams{
		ItemID:     itemID,
		SellerID:   sellerID,
		StartPrice: startPrice,
		EndsAt:     TimeToPgTimestamptz(endsAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create auction row: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ToDomainAuction(a), nil
}

func (r *auctionRepository) ListActive(ctx context.Context, limit, offset int32) ([]*domain.Auction, int64, error) {
	total, err := r.queries.CountActiveAuctions(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count active auctions from db: %w", err)
	}

	rows, err := r.queries.ListActiveAuctions(ctx, sqlc.ListActiveAuctionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list active auctions: %w", err)
	}

	auctions := make([]*domain.Auction, len(rows))
	for i, v := range rows {
		auctions[i] = &domain.Auction{
			ID:         v.ID,
			ItemID:     v.ItemID,
			SellerID:   v.SellerID,
			Status:     domain.AuctionStatus(v.Status),
			StartPrice: v.StartPrice,
			HighestBid: Int8ToPtr(v.HighestBid),
			WinnerID:   UUIDToPtr(v.WinnerID),
			EndsAt:     v.EndsAt.Time,
			CreatedAt:  v.CreatedAt.Time,
		}
	}
	return auctions, total, nil
}

func (r *auctionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Auction, error) {
	a, err := r.queries.GetAuction(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAuctionNotFound
		}
		return nil, fmt.Errorf("failed to get auction by id: %w", err)
	}
	return ToDomainAuction(a), nil
}

func (r *auctionRepository) PlaceBidTransaction(ctx context.Context, auctionID, bidderID uuid.UUID, amount int64, extendedEndTime time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Lock auction sequence to guarantee linear bid evaluations
	auction, err := qtx.GetAuctionForUpdate(ctx, auctionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAuctionNotFound
		}
		return fmt.Errorf("failed to lock auction for update: %w", err)
	}
	if auction.Status != sqlc.AuctionStatusActive || time.Now().After(auction.EndsAt.Time) {
		return ErrAuctionNotActive
	}
	if auction.SellerID == bidderID {
		return ErrBidOnOwnAuction
	}

	// Lock the bidder guild row immediately to prevent overlapping financial commitments
	_, err = qtx.GetGuildByIDForUpdate(ctx, bidderID)
	if err != nil {
		return fmt.Errorf("failed to lock bidder guild: %w", err)
	}

	// Inspect financial liquidity safely now that the guild row is locked
	wallet, err := qtx.GetWalletSummary(ctx, bidderID)
	if err != nil {
		return fmt.Errorf("failed to get wallet summary: %w", err)
	}
	if int64(wallet.AvailableBalance) < amount {
		return ErrInsufficientBalance
	}

	// Release trapped gold from the previous top bidder right away
	if auction.WinnerID.Valid {
		prevWinner := uuid.UUID(auction.WinnerID.Bytes)
		prevBid, err := qtx.GetActiveBidByBidder(ctx, sqlc.GetActiveBidByBidderParams{
			AuctionID: auctionID,
			BidderID:  prevWinner,
		})
		if err == nil {
			_, _ = qtx.DeactivateBid(ctx, prevBid.ID)
			_, _ = qtx.LogWalletTransaction(ctx, sqlc.LogWalletTransactionParams{
				GuildID:     prevWinner,
				Type:        sqlc.TransactionTypeRelease,
				Amount:      prevBid.Amount,
				ReferenceID: UUIDToPgUUID(auctionID),
			})
		}
	}

	// Record the new active bid position
	newBid, err := qtx.CreateBid(ctx, sqlc.CreateBidParams{
		AuctionID: auctionID,
		BidderID:  bidderID,
		Amount:    amount,
	})
	if err != nil {
		return fmt.Errorf("failed to create new bid: %w", err)
	}

	// Update auction standings and set extended timer if applicable
	_, err = qtx.UpdateAuctionBid(ctx, sqlc.UpdateAuctionBidParams{
		ID:         auctionID,
		HighestBid: Int64ToPgInt8(amount),
		WinnerID:   UUIDToPgUUID(bidderID),
		EndsAt:     TimeToPgTimestamptz(extendedEndTime),
	})
	if err != nil {
		return fmt.Errorf("failed to update auction standings: %w", err)
	}

	// Append capital reservation to the audit trail
	_, err = qtx.LogWalletTransaction(ctx, sqlc.LogWalletTransactionParams{
		GuildID:     bidderID,
		Type:        sqlc.TransactionTypeReserve,
		Amount:      -amount,
		ReferenceID: UUIDToPgUUID(newBid.ID),
	})
	if err != nil {
		return fmt.Errorf("failed to log wallet reservation: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *auctionRepository) CancelBidTransaction(ctx context.Context, auctionID, bidderID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	auction, err := qtx.GetAuctionForUpdate(ctx, auctionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAuctionNotFound
		}
		return fmt.Errorf("failed to lock auction for cancel: %w", err)
	}
	if auction.WinnerID.Valid && uuid.UUID(auction.WinnerID.Bytes) == bidderID {
		return ErrRetractLeadingBid
	}

	activeBid, err := qtx.GetActiveBidByBidder(ctx, sqlc.GetActiveBidByBidderParams{
		AuctionID: auctionID,
		BidderID:  bidderID,
	})
	if err != nil {
		return ErrActiveBidNotFound
	}

	_, err = qtx.DeactivateBid(ctx, activeBid.ID)
	if err != nil {
		return fmt.Errorf("failed to deactivate bid: %w", err)
	}

	// Free locked funds immediately upon cancellation
	_, err = qtx.LogWalletTransaction(ctx, sqlc.LogWalletTransactionParams{
		GuildID:     bidderID,
		Type:        sqlc.TransactionTypeRelease,
		Amount:      activeBid.Amount,
		ReferenceID: UUIDToPgUUID(auctionID),
	})
	if err != nil {
		return fmt.Errorf("failed to log wallet release: %w", err)
	}

	return tx.Commit(ctx)
}

// FinalizeExpiredAuction handles the atomic transition of an expired auction.
// It executes cleanly inside a row-locked database transaction boundary.
func (r *auctionRepository) FinalizeExpiredAuction(ctx context.Context, auctionID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// 1. Lock the specific target auction to execute serialization guards safely
	auction, err := qtx.GetAuctionForUpdate(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("failed to lock auction for finalization: %w", err)
	}

	// Guard clause: if already completed or cancelled by another thread event, exit early
	if auction.Status != sqlc.AuctionStatusActive {
		return nil
	}

	// Case A: The auction expired with zero active bids submitted
	if !auction.HighestBid.Valid {
		_, err = qtx.UpdateItemStatus(ctx, sqlc.UpdateItemStatusParams{
			ID:     auction.ItemID,
			Status: sqlc.ItemStatusAvailable, // Rollback item back to general catalog availability
		})
		if err != nil {
			return fmt.Errorf("failed to rollback item status to available: %w", err)
		}

		_, err = qtx.CloseAuction(ctx, sqlc.CloseAuctionParams{
			ID:     auctionID,
			Status: sqlc.AuctionStatusCancelled,
		})
		if err != nil {
			return fmt.Errorf("failed to close auction as cancelled: %w", err)
		}

		return tx.Commit(ctx)
	}

	// Case B: Auction has a finalized frontrunner winner
	winnerID := uuid.UUID(auction.WinnerID.Bytes)
	_, err = qtx.TransferItemOwnership(ctx, sqlc.TransferItemOwnershipParams{
		ID:      auction.ItemID,
		OwnerID: winnerID,
		Status:  sqlc.ItemStatusSold,
	})
	if err != nil {
		return fmt.Errorf("failed to transfer item ownership to winner: %w", err)
	}

	// Dispatch locked reservation funds directly to the vendor/seller guild treasury balance
	_, err = qtx.UpdateGuildBalance(ctx, sqlc.UpdateGuildBalanceParams{
		ID:          auction.SellerID,
		GoldBalance: auction.HighestBid.Int64,
	})
	if err != nil {
		return fmt.Errorf("failed to update seller guild balance: %w", err)
	}

	// Seal the auction board state permanently as ended
	_, err = qtx.CloseAuction(ctx, sqlc.CloseAuctionParams{
		ID:       auctionID,
		Status:   sqlc.AuctionStatusEnded,
		WinnerID: auction.WinnerID,
	})
	if err != nil {
		return fmt.Errorf("failed to permanently close auction: %w", err)
	}

	return tx.Commit(ctx)
}
