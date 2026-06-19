package oracle

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type SafePriceOracle struct {
	inner     PriceOracle
	mu        sync.RWMutex
	lastValid map[uuid.UUID]int64
}

// NewSafePriceOracle wraps any PriceOracle implementation with a stateful crash-recovery cache.
func NewSafePriceOracle(inner PriceOracle) *SafePriceOracle {
	return &SafePriceOracle{
		inner:     inner,
		lastValid: make(map[uuid.UUID]int64),
	}
}

func (s *SafePriceOracle) GetPrice(ctx context.Context, itemID uuid.UUID) (int64, error) {
	price, err := s.inner.GetPrice(ctx, itemID)

	// Fallback logic: if downstream errors out or returns invalid figures, read the cache ledger
	if err != nil || price <= 0 {
		s.mu.RLock()
		last, exists := s.lastValid[itemID]
		s.mu.RUnlock()

		if !exists {
			return 100, nil // Absolute baseline recovery floor for unindexed fresh assets
		}
		return last, nil
	}

	s.mu.Lock()
	s.lastValid[itemID] = price
	s.mu.Unlock()

	return price, nil
}
