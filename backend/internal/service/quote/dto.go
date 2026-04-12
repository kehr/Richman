package quote

import "time"

// QuoteDTO is the HTTP response payload for the quote endpoint.
type QuoteDTO struct {
	AssetCode string         `json:"assetCode"`
	AssetType string         `json:"assetType"`
	Source    string         `json:"source"`
	FetchedAt time.Time      `json:"fetchedAt"`
	Current   *CurrentQuote  `json:"current"`
	History   []HistoryPoint `json:"history"`
}

// CurrentQuote holds the latest price and daily change.
type CurrentQuote struct {
	Price     float64   `json:"price"`
	Date      time.Time `json:"date"`
	ChangeAbs float64   `json:"changeAbs"`
	ChangePct float64   `json:"changePct"`
}

// HistoryPoint holds a single OHLCV bar.
type HistoryPoint struct {
	Date   time.Time `json:"date"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}
