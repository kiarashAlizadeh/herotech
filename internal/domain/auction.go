package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuctionStatus string

const (
	AuctionStatusActive    AuctionStatus = "active"
	AuctionStatusEnded     AuctionStatus = "ended"
	AuctionStatusCancelled AuctionStatus = "cancelled"
)

// Auction governs the live bidding process for legendary, one-of-a-kind artifacts.
type Auction struct {
	ID         uuid.UUID
	ItemID     uuid.UUID
	SellerID   uuid.UUID
	Status     AuctionStatus
	StartPrice int64
	HighestBid *int64     // Nil if no bids have been submitted yet
	WinnerID   *uuid.UUID // Frontrunner or finalized winner
	EndsAt     time.Time
	CreatedAt  time.Time
}

// Bid represents a capital commitment made by a guild toward a specific live auction.
type Bid struct {
	ID        uuid.UUID
	AuctionID uuid.UUID
	BidderID  uuid.UUID
	Amount    int64
	IsActive  bool // Flaps to false once outbid or retracted
	CreatedAt time.Time
}
