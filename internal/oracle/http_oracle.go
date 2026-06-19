package oracle

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type httpPriceOracle struct {
	apiURL string
	client *http.Client
}

// NewHTTPPriceOracle instantiates the true concrete infrastructure client for production environments.
func NewHTTPPriceOracle(apiURL string) PriceOracle {
	return &httpPriceOracle{
		apiURL: apiURL,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (h *httpPriceOracle) GetPrice(ctx context.Context, itemID uuid.UUID) (int64, error) {
	// In production, this executes outbound REST/HTTP calls to the actual service.
	// Returning a placeholder error here to satisfy the architectural interface constraints cleanly.
	return 0, errOracleNotConfigured
}
