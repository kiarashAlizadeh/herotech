package domain

import (
	"time"

	"github.com/google/uuid"
)

type ItemType string

const (
	ItemTypeCommon    ItemType = "common"
	ItemTypeRare      ItemType = "rare"
	ItemTypeLegendary ItemType = "legendary"
)

type ItemStatus string

const (
	ItemStatusAvailable ItemStatus = "available"
	ItemStatusInAuction ItemStatus = "in_auction"
	ItemStatusSold      ItemStatus = "sold"
)

// Item represents a magical artifact traded in Aethoria.
// Depending on its scarcity type, it follows distinct trading flows (limit orders vs auctions).
type Item struct {
	ID        uuid.UUID
	Name      string
	Type      ItemType
	Status    ItemStatus
	OwnerID   uuid.UUID
	BasePrice int64
	ListPrice *int64 // Set only for common/rare assets
	CreatedAt time.Time
	UpdatedAt time.Time
}
