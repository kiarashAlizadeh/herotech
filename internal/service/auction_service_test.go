package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/mocks"
	"github.com/kiarashAlizadeh/herotech/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuctionService_PlaceBid(t *testing.T) {
	auctionID := uuid.New()
	bidderID := uuid.New()
	currentBid := int64(1000)

	tests := []struct {
		name       string
		amount     int64
		setupMock  func(*mocks.AuctionRepository)
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:   "rejects bid below five percent increment",
			amount: 1020,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(&domain.Auction{
					ID:         auctionID,
					StartPrice: 500,
					HighestBid: &currentBid,
					EndsAt:     time.Now().Add(time.Hour),
				}, nil).Once()
			},
			wantErr: ErrBidTooLow,
		},
		{
			name:   "accepts valid bid above current standing",
			amount: 1100,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(&domain.Auction{
					ID:         auctionID,
					StartPrice: 500,
					HighestBid: &currentBid,
					EndsAt:     time.Now().Add(time.Hour),
				}, nil).Once()
				repo.On("PlaceBidTransaction", mock.Anything, auctionID, bidderID, int64(1100), mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
		},
		{
			name:   "accepts first bid at start price",
			amount: 500,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(&domain.Auction{
					ID:         auctionID,
					StartPrice: 500,
					EndsAt:     time.Now().Add(time.Hour),
				}, nil).Once()
				repo.On("PlaceBidTransaction", mock.Anything, auctionID, bidderID, int64(500), mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
		},
		{
			name:   "maps missing auction",
			amount: 1100,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(nil, repository.ErrAuctionNotFound).Once()
			},
			wantErr: ErrAuctionNotFound,
		},
		{
			name:   "returns internal error for lookup failure",
			amount: 1100,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(nil, errors.New("db down")).Once()
			},
			wantAnyErr: true,
		},
		{
			name:   "extends auction when bid arrives near deadline",
			amount: 1100,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(&domain.Auction{
					ID:         auctionID,
					StartPrice: 500,
					HighestBid: &currentBid,
					EndsAt:     time.Now().Add(time.Minute),
				}, nil).Once()
				repo.On("PlaceBidTransaction", mock.Anything, auctionID, bidderID, int64(1100), mock.MatchedBy(func(extended time.Time) bool {
					return math.Abs(time.Until(extended).Seconds()-300) < 1
				})).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewAuctionRepository(t)
			tt.setupMock(repo)

			svc := NewAuctionService(repo, nil)
			err := svc.PlaceBid(context.Background(), auctionID, bidderID, dto.PlaceBidRequest{Amount: tt.amount})

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.wantAnyErr {
				require.Error(t, err)
				assert.NotErrorIs(t, err, ErrAuctionNotFound)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestAuctionService_StartAuction(t *testing.T) {
	sellerID := uuid.New()
	itemID := uuid.New()

	tests := []struct {
		name      string
		req       dto.CreateAuctionRequest
		setupMock func(*mocks.AuctionRepository, *mocks.ItemRepository)
		wantErr   error
	}{
		{
			name: "rejects non legendary item",
			req:  dto.CreateAuctionRequest{ItemID: itemID, StartPrice: 1000, Duration: 2},
			setupMock: func(_ *mocks.AuctionRepository, items *mocks.ItemRepository) {
				items.On("GetByID", mock.Anything, itemID).Return(&domain.Item{
					ID:      itemID,
					Type:    domain.ItemTypeRare,
					OwnerID: sellerID,
				}, nil).Once()
			},
			wantErr: ErrNonLegendaryAuction,
		},
		{
			name: "rejects item owned by another guild",
			req:  dto.CreateAuctionRequest{ItemID: itemID, StartPrice: 1000, Duration: 2},
			setupMock: func(_ *mocks.AuctionRepository, items *mocks.ItemRepository) {
				items.On("GetByID", mock.Anything, itemID).Return(&domain.Item{
					ID:      itemID,
					Type:    domain.ItemTypeLegendary,
					OwnerID: uuid.New(),
				}, nil).Once()
			},
			wantErr: ErrNotItemOwner,
		},
		{
			name: "maps missing associated item",
			req:  dto.CreateAuctionRequest{ItemID: itemID, StartPrice: 1000, Duration: 2},
			setupMock: func(_ *mocks.AuctionRepository, items *mocks.ItemRepository) {
				items.On("GetByID", mock.Anything, itemID).Return(nil, repository.ErrItemNotFound).Once()
			},
			wantErr: ErrAssociatedItemNotFound,
		},
		{
			name: "returns auction response on success",
			req:  dto.CreateAuctionRequest{ItemID: itemID, StartPrice: 1000, Duration: 2},
			setupMock: func(auctions *mocks.AuctionRepository, items *mocks.ItemRepository) {
				items.On("GetByID", mock.Anything, itemID).Return(&domain.Item{
					ID:      itemID,
					Type:    domain.ItemTypeLegendary,
					OwnerID: sellerID,
				}, nil).Once()
				auctions.On("Create", mock.Anything, itemID, sellerID, int64(1000), 2*time.Hour).Return(&domain.Auction{
					ID:         uuid.New(),
					ItemID:     itemID,
					SellerID:   sellerID,
					Status:     domain.AuctionStatusActive,
					StartPrice: 1000,
					EndsAt:     time.Now().Add(2 * time.Hour),
				}, nil).Once()
			},
		},
		{
			name: "wraps repository create failure",
			req:  dto.CreateAuctionRequest{ItemID: itemID, StartPrice: 1000, Duration: 2},
			setupMock: func(auctions *mocks.AuctionRepository, items *mocks.ItemRepository) {
				items.On("On", mock.Anything, itemID).Return(&domain.Item{
					ID:      itemID,
					Type:    domain.ItemTypeLegendary,
					OwnerID: sellerID,
				}, nil).Once()
				auctions.On("Create", mock.Anything, itemID, sellerID, int64(1000), 2*time.Hour).Return(nil, errors.New("insert failed")).Once()
			},
			wantErr: errors.New("insert failed"),
		},
		{
			name: "maps seller active auction limit",
			req:  dto.CreateAuctionRequest{ItemID: itemID, StartPrice: 1000, Duration: 2},
			setupMock: func(auctions *mocks.AuctionRepository, items *mocks.ItemRepository) {
				items.On("GetByID", mock.Anything, itemID).Return(&domain.Item{
					ID:      itemID,
					Type:    domain.ItemTypeLegendary,
					OwnerID: sellerID,
				}, nil).Once()
				auctions.On("Create", mock.Anything, itemID, sellerID, int64(1000), 2*time.Hour).Return(nil, repository.ErrMaxActiveAuctions).Once()
			},
			wantErr: ErrMaxActiveAuctions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auctions := mocks.NewAuctionRepository(t)
			items := mocks.NewItemRepository(t)
			tt.setupMock(auctions, items)

			res, err := NewAuctionService(auctions, items).StartAuction(context.Background(), sellerID, tt.req)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, ErrAssociatedItemNotFound) || errors.Is(tt.wantErr, ErrNonLegendaryAuction) || errors.Is(tt.wantErr, ErrNotItemOwner) {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, itemID, res.ItemID)
			assert.Equal(t, sellerID, res.SellerID)
		})
	}
}

func TestAuctionService_GetAuction(t *testing.T) {
	auctionID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(*mocks.AuctionRepository)
		wantErr   error
	}{
		{
			name:    "rejects nil id",
			id:      uuid.Nil,
			wantErr: ErrInvalidAuctionID,
		},
		{
			name: "maps missing auction",
			id:   auctionID,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(nil, repository.ErrAuctionNotFound).Once()
			},
			wantErr: ErrAuctionNotFound,
		},
		{
			name: "returns internal error for repository failure",
			id:   auctionID,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(nil, errors.New("db down")).Once()
			},
		},
		{
			name: "returns auction",
			id:   auctionID,
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("GetByID", mock.Anything, auctionID).Return(&domain.Auction{
					ID:         auctionID,
					ItemID:     uuid.New(),
					SellerID:   uuid.New(),
					Status:     domain.AuctionStatusActive,
					StartPrice: 500,
					EndsAt:     time.Now().Add(time.Hour),
				}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewAuctionRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			res, err := NewAuctionService(repo, nil).GetAuction(context.Background(), tt.id)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.setupMock == nil || tt.name == "returns internal error for repository failure" {
				require.Error(t, err)
				assert.NotErrorIs(t, err, ErrAuctionNotFound)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, res.ID)
		})
	}
}
