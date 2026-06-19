package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/repository"
)

type GuildService interface {
	CreateGuild(ctx context.Context, req dto.CreateGuildRequest) (*dto.GuildResponse, error)
	ListGuilds(ctx context.Context, req dto.PaginationRequest) (*dto.PaginatedResponse[dto.GuildResponse], error)
	GetGuild(ctx context.Context, id uuid.UUID) (*dto.GuildResponse, error)
	GetGuildInventory(ctx context.Context, id uuid.UUID) (*dto.GuildInventoryResponse, error)
	GetWalletSummary(ctx context.Context, id uuid.UUID) (*dto.WalletSummaryResponse, error)
	DepositGold(ctx context.Context, id uuid.UUID, req dto.DepositGoldRequest) (*dto.GuildResponse, error)
}

type guildService struct {
	repo     repository.GuildRepository
	itemRepo repository.ItemRepository
}

func NewGuildService(repo repository.GuildRepository, itemRepo repository.ItemRepository) GuildService {
	return &guildService{
		repo:     repo,
		itemRepo: itemRepo,
	}
}

func (s *guildService) CreateGuild(ctx context.Context, req dto.CreateGuildRequest) (*dto.GuildResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrEmptyGuildName
	}
	if len(name) < 2 || len(name) > 100 {
		return nil, ErrInvalidGuildNameLength
	}
	if req.DailyLimit <= 0 {
		return nil, ErrInvalidDailyLimit
	}

	g, err := s.repo.Create(ctx, name, req.DailyLimit)
	if err != nil {
		// Catch PostgreSQL unique constraint violation for duplicate guild names safely
		if strings.Contains(err.Error(), "guilds_name_key") || strings.Contains(err.Error(), "23505") {
			return nil, ErrGuildNameExists
		}
		return nil, fmt.Errorf("failed to create guild: %w", err)
	}

	return &dto.GuildResponse{
		ID:          g.ID,
		Name:        g.Name,
		GoldBalance: g.GoldBalance,
		DailyLimit:  g.DailyLimit,
		CreatedAt:   g.CreatedAt,
	}, nil
}

func (s *guildService) ListGuilds(ctx context.Context, req dto.PaginationRequest) (*dto.PaginatedResponse[dto.GuildResponse], error) {
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

	rows, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list guilds from repository: %w", err)
	}

	res := make([]dto.GuildResponse, len(rows))
	for i, g := range rows {
		res[i] = dto.GuildResponse{
			ID:          g.ID,
			Name:        g.Name,
			GoldBalance: g.GoldBalance,
			DailyLimit:  g.DailyLimit,
			CreatedAt:   g.CreatedAt,
		}
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		totalPages = 0
	}

	return &dto.PaginatedResponse[dto.GuildResponse]{
		Data: res,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *guildService) GetGuild(ctx context.Context, id uuid.UUID) (*dto.GuildResponse, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidGuildID
	}

	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		// Map repository error safely to service domain error
		if errors.Is(err, repository.ErrGuildNotFound) {
			return nil, ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to retrieve guild: %w", err)
	}

	return &dto.GuildResponse{
		ID:          g.ID,
		Name:        g.Name,
		GoldBalance: g.GoldBalance,
		DailyLimit:  g.DailyLimit,
		CreatedAt:   g.CreatedAt,
	}, nil
}

func (s *guildService) GetGuildInventory(ctx context.Context, id uuid.UUID) (*dto.GuildInventoryResponse, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidGuildID
	}

	// Verify the target guild actually exists before parsing inventory
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrGuildNotFound) {
			return nil, ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to validate guild container: %w", err)
	}

	// Fetch all raw items associated with this guild identity
	rawItems, err := s.itemRepo.GetByOwner(ctx, id) // Assuming itemRepo is injected into guildService
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw asset ledger: %w", err)
	}

	listed := make([]dto.ItemResponse, 0)
	purchased := make([]dto.ItemResponse, 0)

	// Segment items based on their structural marketplace status flags
	for _, item := range rawItems {
		responseModel := dto.ItemResponse{
			ID:        item.ID,
			Name:      item.Name,
			Type:      string(item.Type),
			Status:    string(item.Status),
			OwnerID:   item.OwnerID,
			BasePrice: item.BasePrice,
			ListPrice: item.ListPrice,
			CreatedAt: item.CreatedAt,
		}

		// If status is available or in_auction, it means the guild is currently selling it
		if item.Status == domain.ItemStatusAvailable || item.Status == domain.ItemStatusInAuction {
			listed = append(listed, responseModel)
		} else if item.Status == domain.ItemStatusSold {
			// If status is sold, since the owner is this guild, it means they purchased it
			purchased = append(purchased, responseModel)
		}
	}

	return &dto.GuildInventoryResponse{
		ListedItems:    listed,
		PurchasedItems: purchased,
	}, nil
}

func (s *guildService) GetWalletSummary(ctx context.Context, id uuid.UUID) (*dto.WalletSummaryResponse, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidGuildID
	}

	total, reserved, available, err := s.repo.GetWallet(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrGuildNotFound) {
			return nil, ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to fetch wallet details: %w", err)
	}

	return &dto.WalletSummaryResponse{
		TotalBalance:     total,
		ReservedAmount:   reserved,
		AvailableBalance: available,
	}, nil
}

func (s *guildService) DepositGold(ctx context.Context, id uuid.UUID, req dto.DepositGoldRequest) (*dto.GuildResponse, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidGuildID
	}
	if req.Amount <= 0 {
		return nil, ErrInvalidDepositAmount
	}

	g, err := s.repo.DepositGold(ctx, id, req.Amount)
	if err != nil {
		if errors.Is(err, repository.ErrGuildNotFound) {
			return nil, ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to deposit gold asset: %w", err)
	}

	return &dto.GuildResponse{
		ID:          g.ID,
		Name:        g.Name,
		GoldBalance: g.GoldBalance,
		DailyLimit:  g.DailyLimit,
		CreatedAt:   g.CreatedAt,
	}, nil
}
