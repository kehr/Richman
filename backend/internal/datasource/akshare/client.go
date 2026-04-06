package akshare

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/richman/backend/internal/datasource"
	"go.uber.org/zap"
)

// Client talks to an AKShare HTTP proxy for Chinese market data.
// TODO: The production setup requires a Python sidecar running AKShare's HTTP interface.
// For MVP, all methods return realistic mock data so downstream consumers can develop against it.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// New creates a new AKShare client.
func New(baseURL string, logger *zap.Logger) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: datasource.NewHTTPClient(),
		logger:     logger,
	}
}

// FetchPriceHistory returns daily OHLCV data for a Chinese ETF or index.
// TODO: Replace mock data with real AKShare HTTP proxy calls once the Python sidecar is deployed.
func (c *Client) FetchPriceHistory(ctx context.Context, code string, days int) ([]datasource.PriceData, error) {
	c.logger.Info("fetching price history from akshare (mock)", zap.String("code", code), zap.Int("days", days))

	if days <= 0 {
		days = 30
	}

	prices := make([]datasource.PriceData, 0, days)
	basePrice := 3200.0 // approximate CSI 300 level
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		// Skip weekends for realism.
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			continue
		}
		noise := math.Sin(float64(i)*0.3) * 50
		closePrice := basePrice + noise
		prices = append(prices, datasource.PriceData{
			Date:   date.Truncate(24 * time.Hour),
			Open:   closePrice - 10 + math.Sin(float64(i))*5,
			High:   closePrice + 20,
			Low:    closePrice - 15,
			Close:  closePrice,
			Volume: 1_500_000_000 + math.Sin(float64(i))*500_000_000,
		})
	}

	return prices, nil
}

// FetchValuation returns PE/PB valuation data for a Chinese index.
// TODO: Replace mock data with real AKShare HTTP proxy calls once the Python sidecar is deployed.
func (c *Client) FetchValuation(ctx context.Context, code string) (*datasource.ValuationData, error) {
	c.logger.Info("fetching valuation from akshare (mock)", zap.String("code", code))

	return &datasource.ValuationData{
		Date:          time.Now().Truncate(24 * time.Hour),
		PE:            12.5,
		PB:            1.35,
		CAPE:          14.2,
		DividendYield: 0.028,
	}, nil
}
