package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kiarashAlizadeh/herotech/internal/db/sqlc"
	"github.com/kiarashAlizadeh/herotech/internal/domain"
)

type GuildRepository interface {
	Create(ctx context.Context, name string, dailyLimit int64) (*domain.Guild, error)
	List(ctx context.Context, limit, offset int32) ([]*domain.Guild, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Guild, error)
	GetWallet(ctx context.Context, id uuid.UUID) (total, reserved, available int64, err error)
	DepositGold(ctx context.Context, id uuid.UUID, amount int64) (*domain.Guild, error)
}

type guildRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewGuildRepository(db *pgxpool.Pool, queries *sqlc.Queries) GuildRepository {
	return &guildRepository{db: db, queries: queries}
}

func (r *guildRepository) Create(ctx context.Context, name string, dailyLimit int64) (*domain.Guild, error) {
	g, err := r.queries.CreateGuild(ctx, sqlc.CreateGuildParams{
		Name:       name,
		DailyLimit: dailyLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create guild in db: %w", err)
	}
	return ToDomainGuild(g), nil
}

func (r *guildRepository) List(ctx context.Context, limit, offset int32) ([]*domain.Guild, int64, error) {
	total, err := r.queries.CountGuilds(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count guilds from db: %w", err)
	}

	rows, err := r.queries.ListGuilds(ctx, sqlc.ListGuildsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list guilds from db: %w", err)
	}
	res := make([]*domain.Guild, len(rows))
	for i, row := range rows {
		res[i] = &domain.Guild{
			ID:          row.ID,
			Name:        row.Name,
			GoldBalance: row.GoldBalance,
			DailyLimit:  row.DailyLimit,
			CreatedAt:   row.CreatedAt.Time,
		}
	}
	return res, total, nil
}

func (r *guildRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Guild, error) {
	g, err := r.queries.GetGuildByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to get guild by id: %w", err)
	}
	return ToDomainGuild(g), nil
}

func (r *guildRepository) GetWallet(ctx context.Context, id uuid.UUID) (int64, int64, int64, error) {
	w, err := r.queries.GetWalletSummary(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, 0, ErrGuildNotFound
		}
		return 0, 0, 0, fmt.Errorf("failed to get wallet summary: %w", err)
	}
	// sqlc infers SUM/COALESCE as int32, casting to int64 for our domain
	return int64(w.TotalBalance), int64(w.ReservedAmount), int64(w.AvailableBalance), nil
}

func (r *guildRepository) DepositGold(ctx context.Context, id uuid.UUID, amount int64) (*domain.Guild, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin deposit transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Lock the guild row early to prevent balance updates overlapping
	_, err = qtx.GetGuildByIDForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGuildNotFound
		}
		return nil, fmt.Errorf("failed to lock guild for update: %w", err)
	}

	g, err := qtx.UpdateGuildBalance(ctx, sqlc.UpdateGuildBalanceParams{
		ID:          id,
		GoldBalance: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update guild balance: %w", err)
	}

	// Always append to the ledger for transparency
	_, err = qtx.LogWalletTransaction(ctx, sqlc.LogWalletTransactionParams{
		GuildID: id,
		Type:    sqlc.TransactionTypeDeposit,
		Amount:  amount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to log deposit transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit deposit transaction: %w", err)
	}

	return ToDomainGuild(g), nil
}
