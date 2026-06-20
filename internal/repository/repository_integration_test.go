//go:build integration
// +build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kiarashAlizadeh/herotech/internal/db/sqlc"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type repositoryHarness struct {
	pool     *pgxpool.Pool
	queries  *sqlc.Queries
	guilds   GuildRepository
	items    ItemRepository
	auctions AuctionRepository
}

func setupRepositoryHarness(t *testing.T) *repositoryHarness {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("skipping integration tests because Docker is not available: %s", fmt.Sprint(r))
		}
	}()

	ctx := context.Background()
	scripts := migrationScripts(t)

	container, err := postgres.Run(
		ctx,
		"postgres:18.4-alpine3.24",
		postgres.WithDatabase("herotech_test"),
		postgres.WithUsername("herotech"),
		postgres.WithPassword("herotech"),
		postgres.WithOrderedInitScripts(scripts...),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	if err != nil && isDockerUnavailable(err) {
		t.Skipf("skipping integration tests because Docker is not available: %v", err)
	}
	require.NoError(t, err)

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	queries := sqlc.New(pool)
	return &repositoryHarness{
		pool:     pool,
		queries:  queries,
		guilds:   NewGuildRepository(pool, queries),
		items:    NewItemRepository(pool, queries),
		auctions: NewAuctionRepository(pool, queries),
	}
}

func isDockerUnavailable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "docker") && (strings.Contains(msg, "Access is denied") || strings.Contains(msg, "daemon") || strings.Contains(msg, "connect"))
}

func migrationScripts(t *testing.T) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("..", "db", "migrations", "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	sort.Strings(files)

	abs := make([]string, 0, len(files))
	for _, file := range files {
		full, err := filepath.Abs(file)
		require.NoError(t, err)
		abs = append(abs, full)
	}
	return abs
}

func TestItemRepository_PurchaseLimitOrderIntegration(t *testing.T) {
	h := setupRepositoryHarness(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(t *testing.T) (itemID, buyerID uuid.UUID)
		wantErr error
		assert  func(t *testing.T, itemID, buyerID uuid.UUID)
	}{
		{
			name: "purchases available item",
			setup: func(t *testing.T) (uuid.UUID, uuid.UUID) {
				seller := createGuild(t, h, "seller-success", 5000, 0)
				buyer := createGuild(t, h, "buyer-success", 5000, 1000)
				seedDailyPurchase(t, h, buyer.ID, 0)
				item := createItem(t, h, "common-success", domain.ItemTypeCommon, seller.ID, 300)
				return item.ID, buyer.ID
			},
			assert: func(t *testing.T, itemID, buyerID uuid.UUID) {
				item, err := h.queries.GetItemByID(ctx, itemID)
				require.NoError(t, err)
				assert.Equal(t, buyerID, item.OwnerID)
				assert.Equal(t, sqlc.ItemStatusSold, item.Status)
			},
		},
		{
			name: "rejects insufficient gold",
			setup: func(t *testing.T) (uuid.UUID, uuid.UUID) {
				seller := createGuild(t, h, "seller-poor-buyer", 5000, 0)
				buyer := createGuild(t, h, "buyer-poor", 5000, 100)
				seedDailyPurchase(t, h, buyer.ID, 0)
				item := createItem(t, h, "common-expensive", domain.ItemTypeCommon, seller.ID, 300)
				return item.ID, buyer.ID
			},
			wantErr: ErrInsufficientGold,
		},
		{
			name: "rejects daily limit overflow",
			setup: func(t *testing.T) (uuid.UUID, uuid.UUID) {
				seller := createGuild(t, h, "seller-limit", 5000, 0)
				buyer := createGuild(t, h, "buyer-limit", 350, 1000)
				seedDailyPurchase(t, h, buyer.ID, 100)
				item := createItem(t, h, "common-limit", domain.ItemTypeCommon, seller.ID, 300)
				return item.ID, buyer.ID
			},
			wantErr: ErrDailyLimitExceeded,
		},
		{
			name: "rejects own item purchase",
			setup: func(t *testing.T) (uuid.UUID, uuid.UUID) {
				owner := createGuild(t, h, "owner-self", 5000, 1000)
				seedDailyPurchase(t, h, owner.ID, 0)
				item := createItem(t, h, "common-self", domain.ItemTypeCommon, owner.ID, 300)
				return item.ID, owner.ID
			},
			wantErr: ErrPurchaseOwnItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			itemID, buyerID := tt.setup(t)

			err := h.items.PurchaseLimitOrder(ctx, itemID, buyerID)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.assert(t, itemID, buyerID)
		})
	}
}

func TestAuctionRepository_BidTransactionsIntegration(t *testing.T) {
	h := setupRepositoryHarness(t)
	ctx := context.Background()

	t.Run("places valid bid", func(t *testing.T) {
		auction, bidder := createAuctionWithBidder(t, h, "valid-bid", 1000)

		err := h.auctions.PlaceBidTransaction(ctx, auction.ID, bidder.ID, 500, time.Now().Add(time.Hour))

		require.NoError(t, err)
		row, err := h.queries.GetAuction(ctx, auction.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(500), row.HighestBid.Int64)
		assert.Equal(t, bidder.ID, uuid.UUID(row.WinnerID.Bytes))
	})

	t.Run("rejects own auction bid", func(t *testing.T) {
		auction, _ := createAuctionWithBidder(t, h, "own-bid", 1000)

		err := h.auctions.PlaceBidTransaction(ctx, auction.ID, auction.SellerID, 500, time.Now().Add(time.Hour))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrBidOnOwnAuction)
	})

	t.Run("rejects insufficient balance", func(t *testing.T) {
		auction, bidder := createAuctionWithBidder(t, h, "insufficient-bid", 100)

		err := h.auctions.PlaceBidTransaction(ctx, auction.ID, bidder.ID, 500, time.Now().Add(time.Hour))

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInsufficientBalance)
	})
}

func TestAuctionRepository_ConcurrentBidReservationsIntegration(t *testing.T) {
	h := setupRepositoryHarness(t)
	ctx := context.Background()

	sellerA := createGuild(t, h, "seller-concurrent-a", 5000, 0)
	sellerB := createGuild(t, h, "seller-concurrent-b", 5000, 0)
	bidder := createGuild(t, h, "bidder-concurrent", 5000, 1000)
	auctionA := createAuctionForSeller(t, h, "auction-concurrent-a", sellerA.ID)
	auctionB := createAuctionForSeller(t, h, "auction-concurrent-b", sellerB.ID)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, auctionID := range []uuid.UUID{auctionA.ID, auctionB.ID} {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			<-start
			errs <- h.auctions.PlaceBidTransaction(ctx, id, bidder.ID, 800, time.Now().Add(time.Hour))
		}(auctionID)
	}

	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	insufficient := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, ErrInsufficientBalance) {
			insufficient++
		}
	}

	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, insufficient)
}

func createGuild(t *testing.T, h *repositoryHarness, name string, dailyLimit, gold int64) *domain.Guild {
	t.Helper()

	guild, err := h.guilds.Create(context.Background(), name+"-"+uuid.NewString(), dailyLimit)
	require.NoError(t, err)
	if gold > 0 {
		guild, err = h.guilds.DepositGold(context.Background(), guild.ID, gold)
		require.NoError(t, err)
	}
	return guild
}

func createItem(t *testing.T, h *repositoryHarness, name string, itemType domain.ItemType, ownerID uuid.UUID, listPrice int64) *domain.Item {
	t.Helper()

	// 1. Mint the fresh item safely using the 5-arguments Create signature
	item, err := h.items.Create(context.Background(), name+"-"+uuid.NewString(), itemType, ownerID, 100)
	require.NoError(t, err)

	// 2. 🛠️ FIX ONLY INSIDE TEST: Force the test container via raw SQL to expose this item with the requested price.
	// This makes it instantly valid for limit order purchase tests without altering domain source files.
	if listPrice > 0 {
		_, err = h.pool.Exec(context.Background(), `
			UPDATE items 
			SET status = 'available', list_price = $1, updated_at = NOW() 
			WHERE id = $2
		`, listPrice, item.ID)
		require.NoError(t, err)
	}
	return item
}

func createAuctionWithBidder(t *testing.T, h *repositoryHarness, name string, bidderGold int64) (*domain.Auction, *domain.Guild) {
	t.Helper()

	seller := createGuild(t, h, "seller-"+name, 5000, 0)
	bidder := createGuild(t, h, "bidder-"+name, 5000, bidderGold)
	return createAuctionForSeller(t, h, name, seller.ID), bidder
}

func createAuctionForSeller(t *testing.T, h *repositoryHarness, name string, sellerID uuid.UUID) *domain.Auction {
	t.Helper()

	item := createItem(t, h, "legendary-"+name, domain.ItemTypeLegendary, sellerID, 0)
	auction, err := h.auctions.Create(context.Background(), item.ID, sellerID, 100, time.Hour)
	require.NoError(t, err)
	return auction
}

func seedDailyPurchase(t *testing.T, h *repositoryHarness, guildID uuid.UUID, totalSpent int64) {
	t.Helper()

	_, err := h.pool.Exec(context.Background(), `
        INSERT INTO daily_purchases (guild_id, date, total_spent)
        VALUES ($1, CURRENT_DATE, $2)
        ON CONFLICT (guild_id, date) DO UPDATE SET total_spent = EXCLUDED.total_spent
    `, guildID, totalSpent)
	require.NoError(t, err)
}
