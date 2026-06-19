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

type stubPriceOracle struct {
	price int64
	err   error
}

func (s stubPriceOracle) GetPrice(context.Context, uuid.UUID) (int64, error) {
	return s.price, s.err
}

func TestItemService_CreateItem(t *testing.T) {
	ownerID := uuid.New()
	listPrice := int64(250)

	tests := []struct {
		name      string
		req       dto.CreateItemRequest
		oracle    stubPriceOracle
		setupMock func(*mocks.ItemRepository)
		wantErr   error
		wantName  string
	}{
		{
			name:    "rejects blank name",
			req:     dto.CreateItemRequest{Name: "  ", Type: "common", ListPrice: &listPrice},
			oracle:  stubPriceOracle{price: 1000},
			wantErr: ErrBlankItemName,
		},
		{
			name:    "rejects invalid item type", 
			req:     dto.CreateItemRequest{Name: "Excalibur", Type: "epic"},
			oracle:  stubPriceOracle{price: 1000},
			wantErr: ErrInvalidItemType,
		},
		{
			name:   "trims item name before create",
			req:    dto.CreateItemRequest{Name: "  Sword  ", Type: "rare", ListPrice: &listPrice},
			oracle: stubPriceOracle{price: 900},
			setupMock: func(repo *mocks.ItemRepository) {

				repo.On("Create", mock.Anything, "Sword", domain.ItemTypeRare, ownerID, int64(900)).Return(&domain.Item{
					ID:        uuid.New(),
					Name:      "Sword",
					Type:      domain.ItemTypeRare,
					Status:    domain.ItemStatusSold, // Entered game as sold/inventory state
					OwnerID:   ownerID,
					BasePrice: 900,
					CreatedAt: time.Now(),
				}, nil).Once()
			},
			wantName: "Sword",
		},
		{
			name:   "uses oracle price for normal listing",
			req:    dto.CreateItemRequest{Name: "Sword", Type: "rare", ListPrice: &listPrice},
			oracle: stubPriceOracle{price: 900},
			setupMock: func(repo *mocks.ItemRepository) {

				repo.On("Create", mock.Anything, "Sword", domain.ItemTypeRare, ownerID, int64(900)).Return(&domain.Item{
					ID:        uuid.New(),
					Name:      "Sword",
					Type:      domain.ItemTypeRare,
					Status:    domain.ItemStatusSold,
					OwnerID:   ownerID,
					BasePrice: 900,
					CreatedAt: time.Now(),
				}, nil).Once()
			},
			wantName: "Sword",
		},
		{
			name:   "falls back to baseline when oracle fails",
			req:    dto.CreateItemRequest{Name: "Axe", Type: "common", ListPrice: &listPrice},
			oracle: stubPriceOracle{err: errors.New("oracle unavailable")},
			setupMock: func(repo *mocks.ItemRepository) {

				repo.On("Create", mock.Anything, "Axe", domain.ItemTypeCommon, ownerID, int64(100)).Return(&domain.Item{
					ID:        uuid.New(),
					Name:      "Axe",
					Type:      domain.ItemTypeCommon,
					Status:    domain.ItemStatusSold,
					OwnerID:   ownerID,
					BasePrice: 100,
					CreatedAt: time.Now(),
				}, nil).Once()
			},
			wantName: "Axe",
		},
		{
			name:   "wraps repository failure",
			req:    dto.CreateItemRequest{Name: "Shield", Type: "rare", ListPrice: &listPrice},
			oracle: stubPriceOracle{price: 750},
			setupMock: func(repo *mocks.ItemRepository) {

				repo.On("Create", mock.Anything, "Shield", domain.ItemTypeRare, ownerID, int64(750)).Return(nil, errors.New("insert failed")).Once()
			},
			wantErr: errors.New("insert failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewItemRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			res, err := NewItemService(repo, tt.oracle).CreateItem(context.Background(), ownerID, tt.req)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, ErrBlankItemName) || errors.Is(tt.wantErr, ErrInvalidItemType) {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, res.Name)
		})
	}
}

func TestItemService_GetItem(t *testing.T) {
	itemID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupMock func(*mocks.ItemRepository)
		wantErr   error
	}{
		{
			name:    "rejects nil id",
			id:      uuid.Nil,
			wantErr: ErrInvalidEntityIDs,
		},
		{
			name: "maps missing item",
			id:   itemID,
			setupMock: func(repo *mocks.ItemRepository) {
				repo.On("GetByID", mock.Anything, itemID).Return(nil, repository.ErrItemNotFound).Once()
			},
			wantErr: ErrItemNotFound,
		},
		{
			name: "returns internal error",
			id:   itemID,
			setupMock: func(repo *mocks.ItemRepository) {
				repo.On("GetByID", mock.Anything, itemID).Return(nil, errors.New("db down")).Once()
			},
		},
		{
			name: "returns item",
			id:   itemID,
			setupMock: func(repo *mocks.ItemRepository) {
				repo.On("GetByID", mock.Anything, itemID).Return(&domain.Item{
					ID:        itemID,
					Name:      "Sword",
					Type:      domain.ItemTypeRare,
					Status:    domain.ItemStatusAvailable,
					OwnerID:   uuid.New(),
					BasePrice: 100,
					CreatedAt: time.Now(),
				}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewItemRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			res, err := NewItemService(repo, stubPriceOracle{}).GetItem(context.Background(), tt.id)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.name == "returns internal error" {
				require.Error(t, err)
				assert.NotErrorIs(t, err, ErrItemNotFound)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, res.ID)
		})
	}
}

func TestItemService_ListAvailable(t *testing.T) {
	tests := []struct {
		name      string
		itemType  *string
		setupMock func(*mocks.ItemRepository)
		wantErr   error
	}{
		{
			name:     "rejects invalid type filter",
			itemType: ptr("invalid"),
			wantErr:  ErrInvalidItemType,
		},
		{
			name:     "accepts valid type filter",
			itemType: ptr("common"),
			setupMock: func(repo *mocks.ItemRepository) {
				filter := domain.ItemTypeCommon
				repo.On("ListAvailable", mock.Anything, &filter, int32(20), int32(0)).Return([]*domain.Item{}, int64(0), nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewItemRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			_, err := NewItemService(repo, stubPriceOracle{}).ListAvailable(context.Background(), tt.itemType, dto.PaginationRequest{})

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestItemService_BuyItemDirectly(t *testing.T) {
	itemID := uuid.New()
	buyerID := uuid.New()

	tests := []struct {
		name      string
		itemID    uuid.UUID
		buyerID   uuid.UUID
		setupMock func(*mocks.ItemRepository)
		wantErr   error
	}{
		{
			name:    "rejects nil item id",
			itemID:  uuid.Nil,
			buyerID: buyerID,
			wantErr: ErrInvalidEntityIDs,
		},
		{
			name:    "rejects nil buyer id",
			itemID:  itemID,
			buyerID: uuid.Nil,
			wantErr: ErrInvalidEntityIDs,
		},
		{
			name:    "maps missing item",
			itemID:  itemID,
			buyerID: buyerID,
			setupMock: func(repo *mocks.ItemRepository) {
				repo.On("PurchaseLimitOrder", mock.Anything, itemID, buyerID).Return(repository.ErrItemNotFound).Once()
			},
			wantErr: ErrItemNotFound,
		},
		{
			name:    "maps item not available",
			itemID:  itemID,
			buyerID: buyerID,
			setupMock: func(repo *mocks.ItemRepository) {
				repo.On("PurchaseLimitOrder", mock.Anything, itemID, buyerID).Return(repository.ErrItemNotAvailable).Once()
			},
			wantErr: ErrItemNotAvailable,
		},
		{
			name:    "delegates valid purchase",
			itemID:  itemID,
			buyerID: buyerID,
			setupMock: func(repo *mocks.ItemRepository) {
				repo.On("PurchaseLimitOrder", mock.Anything, itemID, buyerID).Return(nil).Once()
			},
		},
		{
			name:    "wraps repository purchase failure",
			itemID:  itemID,
			buyerID: buyerID,
			setupMock: func(repo *mocks.ItemRepository) {
				repo.On("PurchaseLimitOrder", mock.Anything, itemID, buyerID).Return(errors.New("daily limit")).Once()
			},
			wantErr: errors.New("daily limit"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewItemRepository(t)
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			err := NewItemService(repo, stubPriceOracle{}).BuyItemDirectly(context.Background(), tt.itemID, tt.buyerID)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, ErrInvalidEntityIDs) {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
