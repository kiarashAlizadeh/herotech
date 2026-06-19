package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
	"github.com/kiarashAlizadeh/herotech/internal/mocks"
	"github.com/stretchr/testify/mock"
)

func TestAuctionWorker_ProcessExpiredAuctions(t *testing.T) {
	expiredID := uuid.New()
	activeID := uuid.New()

	tests := []struct {
		name      string
		setupMock func(*mocks.AuctionRepository)
	}{
		{
			name: "finalizes only expired auctions",
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("ListActive", mock.Anything, mock.Anything, mock.Anything).Return([]*domain.Auction{
					{ID: expiredID, EndsAt: time.Now().Add(-time.Minute)},
					{ID: activeID, EndsAt: time.Now().Add(time.Minute)},
				}, int64(2), nil).Once()
				repo.On("FinalizeExpiredAuction", mock.Anything, expiredID).Return(nil).Once()
			},
		},
		{
			name: "tolerates list errors",
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("ListActive", mock.Anything, mock.Anything, mock.Anything).Return(nil, int64(0), errors.New("db down")).Once()
			},
		},
		{
			name: "continues when finalization fails",
			setupMock: func(repo *mocks.AuctionRepository) {
				repo.On("ListActive", mock.Anything, mock.Anything, mock.Anything).Return([]*domain.Auction{
					{ID: expiredID, EndsAt: time.Now().Add(-time.Minute)},
				}, int64(1), nil).Once()
				repo.On("FinalizeExpiredAuction", mock.Anything, expiredID).Return(errors.New("close failed")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewAuctionRepository(t)
			tt.setupMock(repo)

			NewAuctionWorker(repo).processExpiredAuctions(context.Background())
		})
	}
}
