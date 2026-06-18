package repository

import (
	"time"

	"github.com/kiarashAlizadeh/herotech/internal/db/sqlc"
	"github.com/kiarashAlizadeh/herotech/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ============================================================================
// Primitive Primitive Helpers (Only what Dragon Market actually uses)
// ============================================================================

// Int8ToPtr converts nullable pgtype.Int8 (BIGINT) to a Go *int64
func Int8ToPtr(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	val := i.Int64
	return &val
}

// UUIDToPtr converts nullable pgtype.UUID to a Go *uuid.UUID
func UUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

// TextToPtr converts nullable pgtype.Text to a Go *string
func TextToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	val := t.String
	return &val
}

// DateToTime converts pgtype.Date to standard time.Time
func DateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

// Int64ToPgInt8 converts a standard int64 to a nullable pgtype.Int8
func Int64ToPgInt8(v int64) pgtype.Int8 {
	if v <= 0 {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: v, Valid: true}
}

// UUIDToPgUUID converts a google/uuid to a valid pgtype.UUID
func UUIDToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// TimeToPgTimestamptz converts a time.Time to a valid pgtype.Timestamptz
func TimeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// ============================================================================
// Domain Model Mappers (Cleans up the Repository layer completely)
// ============================================================================

func ToDomainGuild(g sqlc.Guild) *domain.Guild {
	return &domain.Guild{
		ID:          g.ID,
		Name:        g.Name,
		GoldBalance: g.GoldBalance,
		DailyLimit:  g.DailyLimit,
		CreatedAt:   g.CreatedAt.Time,
		UpdatedAt:   g.UpdatedAt.Time,
	}
}
