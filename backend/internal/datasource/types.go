package datasource

import "time"

// PriceData holds historical price data for an asset.
type PriceData struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// ValuationData holds valuation metrics.
type ValuationData struct {
	Date          time.Time
	PE            float64 // Price-to-Earnings ratio
	PB            float64 // Price-to-Book ratio
	CAPE          float64 // Cyclically Adjusted PE (Shiller PE)
	DividendYield float64
}

// EventProbability holds Polymarket event probability.
type EventProbability struct {
	MarketID    string
	Question    string
	Probability float64 // 0.0 - 1.0
	Volume      float64
	UpdatedAt   time.Time
}

// FetchResult wraps any data fetch result with metadata.
type FetchResult[T any] struct {
	Data      T
	Source    string
	FetchedAt time.Time
	Stale     bool // true if using cached/old data due to fetch failure
}

// AssetData aggregates all fetched data for a single asset.
type AssetData struct {
	Prices    []PriceData
	Valuation *ValuationData
	Events    []EventProbability
	AssetCode string
	AssetType string
	FetchedAt time.Time
}
