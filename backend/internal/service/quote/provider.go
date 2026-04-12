package quote

import (
	"context"
	"time"

	"github.com/richman/backend/internal/datasource"
)

// QuoteProvider abstracts the data source layer for fetching asset quotes.
type QuoteProvider interface {
	FetchQuote(ctx context.Context, req QuoteRequest) (*QuoteSnapshot, error)
}

// QuoteRequest holds the parameters for a quote fetch.
type QuoteRequest struct {
	AssetType string
	AssetCode string
	Days      int
}

// QuoteSnapshot holds raw price data returned by a provider.
type QuoteSnapshot struct {
	Current   PricePoint
	Previous  PricePoint
	History   []datasource.PriceData
	Source    string
	FetchedAt time.Time
}

// PricePoint holds a single date+close pair.
type PricePoint struct {
	Date  time.Time
	Close float64
}
