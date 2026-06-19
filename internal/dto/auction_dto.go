package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateAuctionRequest struct {
	ItemID     uuid.UUID `json:"item_id"`
	StartPrice int64     `json:"start_price"`
	Duration   int       `json:"duration"` // Duration in hours
}

type PlaceBidRequest struct {
	Amount int64 `json:"amount"`
}

type AuctionResponse struct {
	ID         uuid.UUID  `json:"id"`
	ItemID     uuid.UUID  `json:"item_id"`
	SellerID   uuid.UUID  `json:"seller_id"`
	Status     string     `json:"status"`
	StartPrice int64      `json:"start_price"`
	HighestBid *int64     `json:"highest_bid,omitempty"`
	WinnerID   *uuid.UUID `json:"winner_id,omitempty"`
	EndsAt     time.Time  `json:"ends_at"`
}
