package oracle

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type oracleResponse struct {
	price int64
	err   error
}

type sequenceOracle struct {
	mu        sync.Mutex
	responses []oracleResponse
}

func (s *sequenceOracle) GetPrice(context.Context, uuid.UUID) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.responses) == 0 {
		return 0, errors.New("no response")
	}
	next := s.responses[0]
	s.responses = s.responses[1:]
	return next.price, next.err
}

func TestSafePriceOracle_GetPrice(t *testing.T) {
	itemID := uuid.New()

	tests := []struct {
		name      string
		responses []oracleResponse
		calls     int
		want      []int64
	}{
		{
			name:      "returns and caches valid upstream price",
			responses: []oracleResponse{{price: 800}},
			calls:     1,
			want:      []int64{800},
		},
		{
			name: "falls back to cached price on error",
			responses: []oracleResponse{
				{price: 900},
				{err: errors.New("network down")},
			},
			calls: 2,
			want:  []int64{900, 900},
		},
		{
			name: "falls back to cached price on invalid value",
			responses: []oracleResponse{
				{price: 700},
				{price: 0},
			},
			calls: 2,
			want:  []int64{700, 700},
		},
		{
			name:      "uses baseline when no cached price exists",
			responses: []oracleResponse{{err: errors.New("network down")}},
			calls:     1,
			want:      []int64{100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oracle := NewSafePriceOracle(&sequenceOracle{responses: tt.responses})

			got := make([]int64, 0, tt.calls)
			for i := 0; i < tt.calls; i++ {
				price, err := oracle.GetPrice(context.Background(), itemID)
				require.NoError(t, err)
				got = append(got, price)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSafePriceOracle_ConcurrentAccess(t *testing.T) {
	itemID := uuid.New()
	oracle := NewSafePriceOracle(PriceOracleFunc(func(context.Context, uuid.UUID) (int64, error) {
		return 500, nil
	}))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			price, err := oracle.GetPrice(context.Background(), itemID)
			require.NoError(t, err)
			assert.Equal(t, int64(500), price)
		}()
	}
	wg.Wait()
}

func TestHTTPPriceOracle_GetPrice(t *testing.T) {
	price, err := NewHTTPPriceOracle("https://example.test").GetPrice(context.Background(), uuid.New())

	assert.Zero(t, price)
	assert.ErrorIs(t, err, errOracleNotConfigured)
}

type PriceOracleFunc func(context.Context, uuid.UUID) (int64, error)

func (f PriceOracleFunc) GetPrice(ctx context.Context, itemID uuid.UUID) (int64, error) {
	return f(ctx, itemID)
}
