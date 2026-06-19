package domain

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypePurchase TransactionType = "purchase"
	TransactionTypeReserve  TransactionType = "reserve"
	TransactionTypeRelease  TransactionType = "release"
	TransactionTypeRefund   TransactionType = "refund"
)

// WalletTransaction provides an immutable audit ledger entry for any financial shift in the ecosystem.
type WalletTransaction struct {
	ID          uuid.UUID
	GuildID     uuid.UUID
	Type        TransactionType
	Amount      int64      // Positive values inject gold, negative values lock/withdraw gold
	ReferenceID *uuid.UUID // Points to the triggering Bid, Auction, or Item
	Description *string
	CreatedAt   time.Time
}
