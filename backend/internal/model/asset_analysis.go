package model

import (
	"encoding/json"
	"time"
)

// AssetAnalysis maps to rs_asset_analyses (read-only from richman).
// All analysis data is written by richson; richman only reads from this table.
type AssetAnalysis struct {
	AssetAnalysisID      int64
	AssetCode            string
	Locale               string
	OverallScore         float64
	SignalLevel          string
	Confidence           float64
	ConfidenceBandLow    float64
	ConfidenceBandHigh   float64
	ModelVersion         string
	MarketInterpretation string
	RiskFactors          json.RawMessage
	RegimeSummary        string
	D1Score              *float64
	D1BaseScore          *float64
	D1LLMAdjustment      *float64
	D2Score              *float64
	D2BaseScore          *float64
	D2LLMAdjustment      *float64
	D3Score              *float64
	D3BaseScore          *float64
	D3LLMAdjustment      *float64
	D4Score              *float64
	D4BaseScore          *float64
	D4LLMAdjustment      *float64
	D1Weight             float64
	D2Weight             float64
	D3Weight             float64
	D4Weight             float64
	LLMSkipped           bool
	DataCoverage         string
	ConflictType         *string
	ConflictMessage      *string
	PrevAnalysisID       *int64
	ScoreDelta           *float64
	ChangeSummary        *string
	MajorChangeRecap     *string
	DataSnapshotAt       time.Time
	UsdExchangeRate      *float64        // CNY/USD snapshot; NULL for USD assets
	PriceAtAnalysis      *float64
	DemoPlan             json.RawMessage
	AnalysisMetadata     json.RawMessage // extensible JSONB (drawdownReference, etc.)
	GeneratedBy          string
	Source               string
	JobID                *string
	AnalyzedAt           time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	IsDeleted            int
}

// AnalysisDimension maps to rs_asset_analysis_dimensions (read-only).
// Each row represents one quantitative indicator's contribution to the analysis.
type AnalysisDimension struct {
	ID                int64
	AssetAnalysisID   int64
	Dimension         string
	SubIndicator      string
	RawValue          *float64
	Percentile1Y      *float64
	Percentile5Y      *float64
	BlendedPercentile *float64
	NormalizedScore   *float64
	WeightInDimension *float64
	DataSource        *string
	DataAsOf          *time.Time
}
