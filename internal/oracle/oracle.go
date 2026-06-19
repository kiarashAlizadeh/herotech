package oracle

import (
	"context"

	"github.com/google/uuid"
)

// PriceOracle defines the boundary for fetching real-time asset baselines.
type PriceOracle interface {
	GetPrice(ctx context.Context, itemID uuid.UUID) (int64, error)
}
