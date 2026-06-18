package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/oracle"
	"github.com/kiarashAlizadeh/herotech/internal/repository"
)

type ItemService interface {
	ListItem(ctx context.Context, ownerID uuid.UUID, req dto.CreateItemRequest) (*dto.ItemResponse, error)
	GetItem(ctx context.Context, id uuid.UUID) (*dto.ItemResponse, error)
	ListAvailable(ctx context.Context, itemType *string, req dto.PaginationRequest) (*dto.PaginatedResponse[dto.ItemResponse], error)
	BuyItemDirectly(ctx context.Context, itemID, buyerID uuid.UUID) error
}

type itemService struct {
	repo        repository.ItemRepository
	priceOracle oracle.PriceOracle
}

func NewItemService(repo repository.ItemRepository, po oracle.PriceOracle) ItemService {
	return &itemService{repo: repo, priceOracle: po}
}

func (s *itemService) ListItem(ctx context.Context, ownerID uuid.UUID, req dto.CreateItemRequest) (*dto.ItemResponse, error) {
	if ownerID == uuid.Nil {
		return nil, ErrInvalidEntityIDs
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrBlankItemName
	}
	if len(name) < 2 || len(name) > 100 {
		return nil, ErrInvalidItemNameLength
	}

	iType := domain.ItemType(req.Type)
	if iType != domain.ItemTypeCommon && iType != domain.ItemTypeRare && iType != domain.ItemTypeLegendary {
		return nil, ErrInvalidItemType
	}

	// Fetch dynamic baseline using our fault-tolerant wrapper layout
	basePrice, err := s.priceOracle.GetPrice(ctx, uuid.New())
	if err != nil {
		basePrice = 100 // Hard recovery line if fallback registry is entirely empty
	}

	var listPrice int64
	if iType == domain.ItemTypeCommon || iType == domain.ItemTypeRare {
		if req.ListPrice == nil || *req.ListPrice <= 0 {
			return nil, ErrInvalidListPrice
		}
		listPrice = *req.ListPrice
	} else if iType == domain.ItemTypeLegendary && req.ListPrice != nil {
		return nil, ErrLegendaryPriceAllowed
	}

	i, err := s.repo.Create(ctx, name, iType, ownerID, basePrice, listPrice)
	if err != nil {
		return nil, fmt.Errorf("failed to register asset inside ledger: %w", err)
	}

	return mapItemToResponse(i), nil
}

func (s *itemService) GetItem(ctx context.Context, id uuid.UUID) (*dto.ItemResponse, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidEntityIDs
	}
	i, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrItemNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to fetch item: %w", err)
	}
	return mapItemToResponse(i), nil
}

func (s *itemService) ListAvailable(ctx context.Context, itemType *string, req dto.PaginationRequest) (*dto.PaginatedResponse[dto.ItemResponse], error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	limit := int32(pageSize)
	offset := int32((page - 1) * pageSize)

	var tFilter *domain.ItemType
	if itemType != nil {
		filterValue := strings.TrimSpace(*itemType)
		if filterValue == "" {
			itemType = nil
		} else {
			filter := domain.ItemType(filterValue)
			if filter != domain.ItemTypeCommon && filter != domain.ItemTypeRare && filter != domain.ItemTypeLegendary {
				return nil, ErrInvalidItemType
			}
			tFilter = &filter
		}
	}

	rows, total, err := s.repo.ListAvailable(ctx, tFilter, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list available items from repository: %w", err)
	}

	res := make([]dto.ItemResponse, len(rows))
	for i, v := range rows {
		mapped := mapItemToResponse(v)
		res[i] = *mapped
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		totalPages = 0
	}

	return &dto.PaginatedResponse[dto.ItemResponse]{
		Data: res,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *itemService) BuyItemDirectly(ctx context.Context, itemID, buyerID uuid.UUID) error {
	if itemID == uuid.Nil || buyerID == uuid.Nil {
		return ErrInvalidEntityIDs
	}

	if err := s.repo.PurchaseLimitOrder(ctx, itemID, buyerID); err != nil {
		if errors.Is(err, repository.ErrItemNotFound) {
			return ErrItemNotFound
		}
		if errors.Is(err, repository.ErrItemNotAvailable) {
			return ErrItemNotAvailable
		}
		if errors.Is(err, repository.ErrPurchaseOwnItem) {
			return ErrPurchaseOwnItem
		}
		if errors.Is(err, repository.ErrInsufficientGold) {
			return ErrInsufficientGold
		}
		if errors.Is(err, repository.ErrDailyLimitExceeded) {
			return ErrDailyLimitExceeded
		}
		if errors.Is(err, repository.ErrGuildNotFound) {
			return ErrGuildNotFound
		}
		return fmt.Errorf("failed to execute direct purchase order: %w", err)
	}
	return nil
}

func mapItemToResponse(i *domain.Item) *dto.ItemResponse {
	return &dto.ItemResponse{
		ID:        i.ID,
		Name:      i.Name,
		Type:      string(i.Type),
		Status:    string(i.Status),
		OwnerID:   i.OwnerID,
		BasePrice: i.BasePrice,
		ListPrice: i.ListPrice,
		CreatedAt: i.CreatedAt,
	}
}
