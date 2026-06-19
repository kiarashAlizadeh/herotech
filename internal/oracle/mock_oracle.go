package oracle

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type mockPriceOracle struct {
	rng *rand.Rand
}

// NewMockPriceOracle builds an unstable pricing emulator to simulate chaotic networks.
func NewMockPriceOracle() PriceOracle {
	return &mockPriceOracle{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *mockPriceOracle) GetPrice(ctx context.Context, itemID uuid.UUID) (int64, error) {
	// Simulate unpredictable network latency (up to 120ms)
	time.Sleep(time.Duration(m.rng.Intn(120)) * time.Millisecond)

	// 10% chance the external oracle service drops entirely
	if m.rng.Intn(10) == 0 {
		return 0, errOracleUnreachable
	}

	// 5% chance the oracle returns corrupted values (zero or negative numbers)
	if m.rng.Intn(20) == 0 {
		return 0, errInvalidPrice
	}

	// Return a valid random fallback baseline price
	return int64(m.rng.Intn(1500) + 150), nil
}
