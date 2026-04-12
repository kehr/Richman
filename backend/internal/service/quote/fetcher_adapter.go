package quote

import (
	"context"
	"fmt"
	"time"

	"github.com/richman/backend/internal/datasource"
)

// FetcherAdapter wraps the existing datasource.Fetcher to satisfy QuoteProvider.
type FetcherAdapter struct {
	fetcher *datasource.Fetcher
}

// NewFetcherAdapter creates a new FetcherAdapter.
func NewFetcherAdapter(fetcher *datasource.Fetcher) *FetcherAdapter {
	return &FetcherAdapter{fetcher: fetcher}
}

// FetchQuote fetches quote data for a given asset via the underlying datasource.Fetcher.
func (a *FetcherAdapter) FetchQuote(ctx context.Context, req QuoteRequest) (*QuoteSnapshot, error) {
	data, err := a.fetcher.FetchAssetData(ctx, req.AssetCode, req.AssetType)
	if err != nil {
		return nil, err
	}

	prices := data.Prices
	if len(prices) == 0 {
		return nil, fmt.Errorf("no price data for %s/%s", req.AssetType, req.AssetCode)
	}

	// Trim to requested days if needed.
	days := req.Days
	if days <= 0 {
		days = 45
	}
	if len(prices) > days {
		prices = prices[len(prices)-days:]
	}

	snap := &QuoteSnapshot{
		Current: PricePoint{
			Date:  prices[len(prices)-1].Date,
			Close: prices[len(prices)-1].Close,
		},
		History:   prices,
		Source:    resolveSourceName(req.AssetType, req.AssetCode),
		FetchedAt: time.Now().UTC(),
	}

	if len(prices) >= 2 {
		snap.Previous = PricePoint{
			Date:  prices[len(prices)-2].Date,
			Close: prices[len(prices)-2].Close,
		}
	}

	return snap, nil
}

// resolveSourceName determines which upstream source was used based on asset
// type and code, mirroring the routing logic in datasource.Fetcher.
func resolveSourceName(assetType, code string) string {
	switch assetType {
	case "us_stock":
		return "yahoo"
	case "gold_etf":
		if isNumericCode(code) {
			return "akshare"
		}
		return "yahoo"
	case "a_share_broad", "a_share_industry":
		return "akshare"
	default:
		return "unknown"
	}
}

func isNumericCode(code string) bool {
	if code == "" {
		return false
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return false
		}
	}
	return true
}
