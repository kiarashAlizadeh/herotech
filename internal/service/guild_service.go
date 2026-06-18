package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kiarashAlizadeh/herotech/internal/dto"
	"github.com/kiarashAlizadeh/herotech/internal/repository"
)

type GuildService interface {
	CreateGuild(ctx context.Context, req dto.CreateGuildRequest) (*dto.GuildResponse, error)
	ListGuilds(ctx context.Context, req dto.PaginationRequest) (*dto.PaginatedResponse[dto.GuildResponse], error)
	GetGuild(ctx context.Context, id uuid.UUID) (*dto.GuildResponse, error)
	GetWalletSummary(ctx context.Context, id uuid.UUID) (*dto.WalletSummaryResponse, error)
	DepositGold(ctx context.Context, id uuid.UUID, req dto.DepositGoldRequest) (*dto.GuildResponse, error)
}

type guildService struct {
	repo repository.GuildRepository
}

func NewGuildService(repo repository.GuildRepository) GuildService {
	return &guildService{repo: repo}
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
