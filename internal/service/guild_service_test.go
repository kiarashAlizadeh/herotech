package service

import (
	"context"
	"errors"
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

func TestGuildService_CreateGuild(t *testing.T) {
	tests := []struct {
		name      string
		req       dto.CreateGuildRequest
		setupMock func(*mocks.GuildRepository)
		wantErr   error
		wantName  string
	}{
		{
			name:    "rejects blank name",
			req:     dto.CreateGuildRequest{Name: " ", DailyLimit: 1000},
			wantErr: ErrEmptyGuildName,
		},
		{
			name: "trims guild name before create",
			req:  dto.CreateGuildRequest{Name: "  Mages  ", DailyLimit: 1000},
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("Create", mock.Anything, "Mages", int64(1000)).Return(&domain.Guild{
					ID:         uuid.New(),
					Name:       "Mages",
					DailyLimit: 1000,
					CreatedAt:  time.Now(),
				}, nil).Once()
			},
			wantName: "Mages",
		},
		{
			name: "creates guild",
			req:  dto.CreateGuildRequest{Name: "Mages", DailyLimit: 1000},
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("Create", mock.Anything, "Mages", int64(1000)).Return(&domain.Guild{
					ID:         uuid.New(),
					Name:       "Mages",
					DailyLimit: 1000,
					CreatedAt:  time.Now(),
				}, nil).Once()
			},
			wantName: "Mages",
		},
		{
			name: "wraps create failure",
			req:  dto.CreateGuildRequest{Name: "Mages", DailyLimit: 1000},
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("Create", mock.Anything, "Mages", int64(1000)).Return(nil, errors.New("duplicate")).Once()
			},
			wantErr: errors.New("duplicate"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewGuildRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			res, err := NewGuildService(repo).CreateGuild(context.Background(), tt.req)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, ErrEmptyGuildName) {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, res.Name)
		})
	}
}

func TestGuildService_GetGuild(t *testing.T) {
	guildID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(*mocks.GuildRepository)
		wantErr   error
	}{
		{
			name:    "rejects nil id",
			id:      uuid.Nil,
			wantErr: ErrInvalidGuildID,
		},
		{
			name: "maps repository not found",
			id:   guildID,
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("GetByID", mock.Anything, guildID).Return(nil, repository.ErrGuildNotFound).Once()
			},
			wantErr: ErrGuildNotFound,
		},
		{
			name: "returns guild",
			id:   guildID,
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("GetByID", mock.Anything, guildID).Return(&domain.Guild{
					ID:         guildID,
					Name:       "Rangers",
					DailyLimit: 2000,
					CreatedAt:  time.Now(),
				}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewGuildRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			res, err := NewGuildService(repo).GetGuild(context.Background(), tt.id)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, res.ID)
		})
	}
}

func TestGuildService_DepositGold(t *testing.T) {
	guildID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(*mocks.GuildRepository)
		wantErr   error
	}{
		{
			name:    "rejects nil id",
			id:      uuid.Nil,
			wantErr: ErrInvalidGuildID,
		},
		{
			name: "maps missing guild",
			id:   guildID,
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("DepositGold", mock.Anything, guildID, int64(500)).Return(nil, repository.ErrGuildNotFound).Once()
			},
			wantErr: ErrGuildNotFound,
		},
		{
			name: "returns updated guild",
			id:   guildID,
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("DepositGold", mock.Anything, guildID, int64(500)).Return(&domain.Guild{
					ID:          guildID,
					Name:        "Rangers",
					GoldBalance: 500,
					DailyLimit:  2000,
					CreatedAt:   time.Now(),
				}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewGuildRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			res, err := NewGuildService(repo).DepositGold(context.Background(), tt.id, dto.DepositGoldRequest{Amount: 500})

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(500), res.GoldBalance)
		})
	}
}

func TestGuildService_GetWalletSummary(t *testing.T) {
	guildID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(*mocks.GuildRepository)
		wantErr   error
	}{
		{
			name:    "rejects nil id",
			id:      uuid.Nil,
			wantErr: ErrInvalidGuildID,
		},
		{
			name: "maps missing guild",
			id:   guildID,
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("GetWallet", mock.Anything, guildID).Return(int64(0), int64(0), int64(0), repository.ErrGuildNotFound).Once()
			},
			wantErr: ErrGuildNotFound,
		},
		{
			name: "returns internal error",
			id:   guildID,
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("GetWallet", mock.Anything, guildID).Return(int64(0), int64(0), int64(0), errors.New("db down")).Once()
			},
		},
		{
			name: "returns wallet summary",
			id:   guildID,
			setupMock: func(repo *mocks.GuildRepository) {
				repo.On("GetWallet", mock.Anything, guildID).Return(int64(1000), int64(250), int64(750), nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewGuildRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			res, err := NewGuildService(repo).GetWalletSummary(context.Background(), tt.id)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.name == "returns internal error" {
				require.Error(t, err)
				assert.NotErrorIs(t, err, ErrGuildNotFound)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, int64(750), res.AvailableBalance)
		})
	}
}
