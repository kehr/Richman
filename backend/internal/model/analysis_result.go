package model

import "time"

// AnalysisResultRecord stores the raw analysis data for audit and debugging.
type AnalysisResultRecord struct {
	ResultID  int64     `json:"resultId"`
	UserID    int64     `json:"userId"`
	HoldingID int64     `json:"holdingId"`
	AssetCode string    `json:"assetCode"`
	RawData   string    `json:"rawData"` // JSON blob of full analysis
	CreatedAt time.Time `json:"createdAt"`
}
