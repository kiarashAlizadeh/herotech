package domain

import (
	"time"

	"github.com/google/uuid"
)

// Guild represents an active faction participating in the dragon market.
// Each guild maintains its own treasury and anti-monopoly expenditure caps.
type Guild struct {
	ID          uuid.UUID
	Name        string
	GoldBalance int64
	DailyLimit  int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DailyPurchase tracks the aggregated capital spent by a guild within a single calendar day.
// It is the core mechanism used to prevent single-guild domination over scarce assets.
type DailyPurchase struct {
	GuildID    uuid.UUID
	Date       time.Time
	TotalSpent int64
}
