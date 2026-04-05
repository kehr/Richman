package datasource

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// PriceHistoryFetcher can fetch price history for an asset.
type PriceHistoryFetcher interface {
	FetchPriceHistory(ctx context.Context, code string, days int) ([]PriceData, error)
}

// ValuationFetcher can fetch valuation data for an asset.
type ValuationFetcher interface {
	FetchValuation(ctx context.Context, code string) (*ValuationData, error)
}

// EventFetcher can fetch event probabilities.
type EventFetcher interface {
	FetchMarketProbabilities(ctx context.Context, keywords []string) ([]EventProbability, error)
}

// Fetcher provides a unified interface for fetching asset data from multiple sources.
type Fetcher struct {
	akshare    PriceAndValuationFetcher
	yahoo      PriceHistoryFetcher
	polymarket EventFetcher
	logger     *zap.Logger
}

// PriceAndValuationFetcher combines price and valuation fetching.
type PriceAndValuationFetcher interface {
	PriceHistoryFetcher
	ValuationFetcher
}

// FetcherDeps holds the dependencies for creating a Fetcher.
type FetcherDeps struct {
	AKShare    PriceAndValuationFetcher
	Yahoo      PriceHistoryFetcher
	Polymarket EventFetcher
	Logger     *zap.Logger
}

// NewFetcher creates a unified Fetcher from its sub-clients.
func NewFetcher(deps FetcherDeps) *Fetcher {
	return &Fetcher{
		akshare:    deps.AKShare,
		yahoo:      deps.Yahoo,
		polymarket: deps.Polymarket,
		logger:     deps.Logger,
	}
}

const defaultFetchDays = 90

// FetchAssetData fetches all relevant data for analysis of a given asset.
// It routes to the correct data source based on assetType:
//   - "gold_etf": yahoo for GLD/IAU, akshare for Chinese gold ETFs (code starting with "5")
//   - "a_share_broad", "a_share_industry": akshare
//   - "us_stock": yahoo
func (f *Fetcher) FetchAssetData(ctx context.Context, assetCode, assetType string) (*AssetData, error) {
	f.logger.Info("fetching asset data",
		zap.String("code", assetCode),
		zap.String("type", assetType),
	)

	result := &AssetData{
		AssetCode: assetCode,
		AssetType: assetType,
		FetchedAt: time.Now(),
	}

	// Fetch prices from the appropriate source.
	var priceFetcher PriceHistoryFetcher
	switch assetType {
	case "gold_etf":
		priceFetcher = f.resolveGoldETFFetcher(assetCode)
	case "a_share_broad", "a_share_industry":
		priceFetcher = f.akshare
	case "us_stock":
		priceFetcher = f.yahoo
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAssetType, assetType)
	}

	prices, err := priceFetcher.FetchPriceHistory(ctx, assetCode, defaultFetchDays)
	if err != nil {
		return nil, fmt.Errorf("fetch prices for %s: %w", assetCode, err)
	}
	result.Prices = prices

	// Fetch valuation for A-share assets.
	if assetType == "a_share_broad" || assetType == "a_share_industry" {
		val, err := f.akshare.FetchValuation(ctx, assetCode)
		if err != nil {
			f.logger.Warn("failed to fetch valuation, continuing without it",
				zap.String("code", assetCode),
				zap.Error(err),
			)
		} else {
			result.Valuation = val
		}
	}

	// Fetch macro event probabilities (best-effort).
	events, err := f.polymarket.FetchMarketProbabilities(ctx, []string{"macro", "fed", "recession"})
	if err != nil {
		f.logger.Warn("failed to fetch event probabilities, continuing without them", zap.Error(err))
	} else {
		result.Events = events
	}

	return result, nil
}

// resolveGoldETFFetcher returns yahoo for US gold ETFs, akshare for Chinese ones.
func (f *Fetcher) resolveGoldETFFetcher(code string) PriceHistoryFetcher {
	if code != "" && code[0] == '5' {
		// Chinese gold ETFs start with "5" (e.g., 518880).
		return f.akshare
	}
	return f.yahoo
}
