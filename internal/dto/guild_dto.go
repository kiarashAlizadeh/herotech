package dto

import (
	"time"

	"github.com/google/uuid"
)

type CreateGuildRequest struct {
	Name       string `json:"name"`
	DailyLimit int64  `json:"daily_limit"`
}

type DepositGoldRequest struct {
	Amount int64 `json:"amount"`
}

type GuildResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	GoldBalance int64     `json:"gold_balance"`
	DailyLimit  int64     `json:"daily_limit"`
	CreatedAt   time.Time `json:"created_at"`
}

type WalletSummaryResponse struct {
	TotalBalance     int64 `json:"total_balance"`
	ReservedAmount   int64 `json:"reserved_amount"`
	AvailableBalance int64 `json:"available_balance"`
}

type GuildInventoryResponse struct {
	ListedItems    []ItemResponse `json:"listed_items"`    // Items currently up for sale (available or in auction)
	PurchasedItems []ItemResponse `json:"purchased_items"` // Items successfully bought and owned by the guild
}
