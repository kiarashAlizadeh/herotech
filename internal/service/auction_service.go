package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/repository"
)

type AuctionService interface {
	StartAuction(ctx context.Context, sellerID uuid.UUID, req dto.CreateAuctionRequest) (*dto.AuctionResponse, error)
	GetAuction(ctx context.Context, id uuid.UUID) (*dto.AuctionResponse, error)
	ListActiveAuctions(ctx context.Context, req dto.PaginationRequest) (*dto.PaginatedResponse[dto.AuctionResponse], error)
	PlaceBid(ctx context.Context, auctionID, bidderID uuid.UUID, req dto.PlaceBidRequest) error
	CancelBid(ctx context.Context, auctionID, bidderID uuid.UUID) error
}

type auctionService struct {
	auctionRepo repository.AuctionRepository
	itemRepo    repository.ItemRepository
}

func NewAuctionService(ar repository.AuctionRepository, ir repository.ItemRepository) AuctionService {
	return &auctionService{auctionRepo: ar, itemRepo: ir}
}

func (s *auctionService) StartAuction(ctx context.Context, sellerID uuid.UUID, req dto.CreateAuctionRequest) (*dto.AuctionResponse, error) {
	if sellerID == uuid.Nil || req.ItemID == uuid.Nil {
		return nil, ErrInvalidEntityIDs
	}
	if req.StartPrice <= 0 {
		return nil, ErrInvalidStartPrice
	}
	if req.Duration <= 0 {
		return nil, ErrInvalidDuration
	}

	item, err := s.itemRepo.GetByID(ctx, req.ItemID)
	if err != nil {
		// Map repository clean error to service domain error
		if errors.Is(err, repository.ErrItemNotFound) {
			return nil, ErrAssociatedItemNotFound
		}
		return nil, fmt.Errorf("failed to fetch item for auction: %w", err)
	}

	if item.Type != domain.ItemTypeLegendary {
		return nil, ErrNonLegendaryAuction
	}
	if item.OwnerID != sellerID {
		return nil, ErrNotItemOwner
	}

	duration := time.Duration(req.Duration) * time.Hour
	a, err := s.auctionRepo.Create(ctx, req.ItemID, sellerID, req.StartPrice, duration)
	if err != nil {
		if errors.Is(err, repository.ErrMaxActiveAuctions) {
			return nil, ErrMaxActiveAuctions
		}
		return nil, fmt.Errorf("failed to create auction via repository: %w", err)
	}

	return mapAuctionToResponse(a), nil
}

func (s *auctionService) GetAuction(ctx context.Context, id uuid.UUID) (*dto.AuctionResponse, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidAuctionID
	}

	a, err := s.auctionRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrAuctionNotFound) {
			return nil, ErrAuctionNotFound
		}
		return nil, fmt.Errorf("failed to fetch auction: %w", err)
	}

	return mapAuctionToResponse(a), nil
}

func (s *auctionService) ListActiveAuctions(ctx context.Context, req dto.PaginationRequest) (*dto.PaginatedResponse[dto.AuctionResponse], error) {
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

	rows, total, err := s.auctionRepo.ListActive(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active auctions list: %w", err)
	}

	res := make([]dto.AuctionResponse, len(rows))
	for i, v := range rows {
		mapped := mapAuctionToResponse(v)
		res[i] = *mapped
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		totalPages = 0
	}

	return &dto.PaginatedResponse[dto.AuctionResponse]{
		Data: res,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *auctionService) PlaceBid(ctx context.Context, auctionID, bidderID uuid.UUID, req dto.PlaceBidRequest) error {
	if auctionID == uuid.Nil {
		return ErrInvalidAuctionID
	}
	if bidderID == uuid.Nil {
		return ErrInvalidGuildID
	}
	if req.Amount <= 0 {
		return ErrInvalidBidAmount
	}

	target, err := s.auctionRepo.GetByID(ctx, auctionID)
	if err != nil {
		if errors.Is(err, repository.ErrAuctionNotFound) {
			return ErrAuctionNotFound
		}
		return fmt.Errorf("failed to fetch auction for bid: %w", err)
	}

	// Calculate and enforce the mandatory 5% step minimum increment
	var minBid int64
	if target.HighestBid != nil {
		minBid = int64(math.Ceil(float64(*target.HighestBid) * 1.05))
	} else {
		minBid = target.StartPrice
	}

	if req.Amount < minBid {
		return ErrBidTooLow
	}

	// 5-minute late bid deadline extension rule logic
	now := time.Now()
	extendedTime := target.EndsAt
	if target.EndsAt.Sub(now) <= 5*time.Minute {
		extendedTime = now.Add(5 * time.Minute)
	}

	if err := s.auctionRepo.PlaceBidTransaction(ctx, auctionID, bidderID, req.Amount, extendedTime); err != nil {
		if errors.Is(err, repository.ErrAuctionNotActive) {
			return ErrAuctionNotActive
		}
		if errors.Is(err, repository.ErrBidOnOwnAuction) {
			return ErrBidOnOwnAuction
		}
		if errors.Is(err, repository.ErrInsufficientBalance) {
			return ErrInsufficientBalance
		}
		return fmt.Errorf("failed to complete bid placement transaction: %w", err)
	}
	return nil
}

func (s *auctionService) CancelBid(ctx context.Context, auctionID, bidderID uuid.UUID) error {
	if auctionID == uuid.Nil {
		return ErrInvalidAuctionID
	}
	if bidderID == uuid.Nil {
		return ErrInvalidGuildID
	}
	if err := s.auctionRepo.CancelBidTransaction(ctx, auctionID, bidderID); err != nil {
		if errors.Is(err, repository.ErrRetractLeadingBid) {
			return ErrRetractLeadingBid
		}
		if errors.Is(err, repository.ErrActiveBidNotFound) {
			return ErrActiveBidNotFound
		}
		if errors.Is(err, repository.ErrAuctionNotFound) {
			return ErrAuctionNotFound
		}
		return fmt.Errorf("failed to cancel bid transaction: %w", err)
	}
	return nil
}

func mapAuctionToResponse(a *domain.Auction) *dto.AuctionResponse {
	return &dto.AuctionResponse{
		ID:         a.ID,
		ItemID:     a.ItemID,
		SellerID:   a.SellerID,
		Status:     string(a.Status),
		StartPrice: a.StartPrice,
		HighestBid: a.HighestBid,
		WinnerID:   a.WinnerID,
		EndsAt:     a.EndsAt,
	}
}
