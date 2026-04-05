package polymarket

import (
	"context"
	"net/http"
	"time"

	"github.com/richman/backend/internal/datasource"
	"go.uber.org/zap"
)

// Client fetches event probability data from Polymarket.
// TODO: Integrate with the real Polymarket CLOB API (https://clob.polymarket.com/markets)
// once access restrictions are resolved. For MVP, returns mock data.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// New creates a new Polymarket client.
func New(logger *zap.Logger) *Client {
	return &Client{
		baseURL:    "https://clob.polymarket.com",
		httpClient: datasource.NewHTTPClient(),
		logger:     logger,
	}
}

// FetchMarketProbabilities returns event probabilities matching the given keywords.
// TODO: Replace with real Polymarket API calls. Current implementation returns mock data.
func (c *Client) FetchMarketProbabilities(
	ctx context.Context,
	keywords []string,
) ([]datasource.EventProbability, error) {
	c.logger.Info("fetching market probabilities from polymarket (mock)", zap.Strings("keywords", keywords))

	now := time.Now()

	// Return realistic mock data for common macro event categories.
	events := []datasource.EventProbability{
		{
			MarketID:    "mock-fed-rate-cut-2025",
			Question:    "Will the Fed cut rates before end of 2025?",
			Probability: 0.72,
			Volume:      15_000_000,
			UpdatedAt:   now,
		},
		{
			MarketID:    "mock-sp500-above-6000",
			Question:    "Will S&P 500 close above 6000 by end of 2025?",
			Probability: 0.45,
			Volume:      8_500_000,
			UpdatedAt:   now,
		},
		{
			MarketID:    "mock-us-recession-2025",
			Question:    "Will the US enter a recession in 2025?",
			Probability: 0.18,
			Volume:      12_000_000,
			UpdatedAt:   now,
		},
	}

	return events, nil
}

// FetchMarketByID returns a specific market's probability by its ID.
// TODO: Replace with real Polymarket API calls.
func (c *Client) FetchMarketByID(ctx context.Context, marketID string) (*datasource.EventProbability, error) {
	c.logger.Info("fetching market by ID from polymarket (mock)", zap.String("marketID", marketID))

	return &datasource.EventProbability{
		MarketID:    marketID,
		Question:    "Mock market question for " + marketID,
		Probability: 0.55,
		Volume:      5_000_000,
		UpdatedAt:   time.Now(),
	}, nil
}
