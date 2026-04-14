package richson

import "time"

// ErrorDetail represents the error payload returned by richson.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details []any  `json:"details"`
}

// ErrorResponse is the top-level error envelope from richson.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// LLMConfig carries provider credentials for per-request LLM overrides.
type LLMConfig struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	APIKey   string  `json:"apiKey"`
	APIBase  *string `json:"apiBase,omitempty"`
}

// DataResponse is the generic envelope that wraps all richson success payloads.
type DataResponse[T any] struct {
	Data T `json:"data"`
}

// ---- Jobs ----

// JobSummary is a lightweight job record returned in list or trigger responses.
type JobSummary struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"`
	AssetCode string    `json:"assetCode"`
	CreatedAt time.Time `json:"createdAt"`
}

// StepInfo describes a single pipeline step within a job.
type StepInfo struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	DurationMs *int64 `json:"durationMs,omitempty"`
}

// JobDetailResponse is the full job record returned by GET /jobs/{jobId}.
type JobDetailResponse struct {
	JobID       string     `json:"jobId"`
	AssetCode   string     `json:"assetCode"`
	Status      string     `json:"status"`
	CurrentStep *string    `json:"currentStep,omitempty"`
	Progress    float64    `json:"progress"`
	Steps       []StepInfo `json:"steps"`
	Error       *string    `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// JobResponse is the response body from POST /jobs/analyze-asset.
type JobResponse struct {
	JobID     string    `json:"jobId"`
	Status    string    `json:"status"`
	AssetCode string    `json:"assetCode"`
	CreatedAt time.Time `json:"createdAt"`
}

// BatchJobSkipped records a single asset that was skipped in a batch request.
type BatchJobSkipped struct {
	AssetCode string `json:"assetCode"`
	Reason    string `json:"reason"`
}

// BatchJobResponse is the response body from POST /jobs/batch-analyze.
type BatchJobResponse struct {
	Jobs    []JobSummary      `json:"jobs"`
	Skipped []BatchJobSkipped `json:"skipped"`
}

// ---- Market ----

// IndexSnapshot holds the current state of a single market index.
type IndexSnapshot struct {
	Name          string  `json:"name"`
	Code          string  `json:"code"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"changePercent"`
}

// MarketRegimeResponse is returned by GET /market/regime.
type MarketRegimeResponse struct {
	Regime       string          `json:"regime"`
	RegimeLabel  string          `json:"regimeLabel"`
	Reason       string          `json:"reason"`
	VIX          float64         `json:"vix"`
	T10Y2Y       float64         `json:"t10y2y"`
	CreditSpread float64         `json:"creditSpread"`
	Indices      []IndexSnapshot `json:"indices"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

// OHLCVCandle holds OHLCV data for a single trading period.
type OHLCVCandle struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}

// OHLCVResponse is returned by GET /market/ohlcv/{code}.
type OHLCVResponse struct {
	AssetCode        string        `json:"assetCode"`
	Currency         string        `json:"currency"`
	Period           string        `json:"period"`
	Candles          []OHLCVCandle `json:"candles"`
	SMA200           *float64      `json:"sma200,omitempty"`
	SupportLevels    []float64     `json:"supportLevels"`
	ResistanceLevels []float64     `json:"resistanceLevels"`
}

// ScorePoint holds scores for a single analysis date.
type ScorePoint struct {
	Date         string  `json:"date"`
	OverallScore float64 `json:"overallScore"`
	D1Score      float64 `json:"d1Score"`
	D2Score      float64 `json:"d2Score"`
	D3Score      float64 `json:"d3Score"`
	D4Score      float64 `json:"d4Score"`
	ModelVersion string  `json:"modelVersion"`
}

// VersionChange records a model version transition in the score history.
type VersionChange struct {
	Date        string `json:"date"`
	FromVersion string `json:"fromVersion"`
	ToVersion   string `json:"toVersion"`
	Note        string `json:"note"`
}

// ScoreHistoryResponse is returned by GET /assets/{code}/score-history.
type ScoreHistoryResponse struct {
	AssetCode      string          `json:"assetCode"`
	Scores         []ScorePoint    `json:"scores"`
	VersionChanges []VersionChange `json:"versionChanges"`
}

// ---- Events ----

// EventItem describes a single macro or market event.
// Pointer fields mirror the "T | None" shape emitted by richson
// (richson/src/richson/schemas/events.py::EventItem); using value types
// here would silently collapse JSON null into Go zero values and re-emit
// them as 0 / "", producing wrong semantics on the client (e.g. "0%"
// probability shown for events that actually have no polymarket data).
type EventItem struct {
	Date                 string   `json:"date"`
	Title                string   `json:"title"`
	Category             string   `json:"category"`
	Impact               string   `json:"impact"`
	GoldDirection        *string  `json:"goldDirection"`
	Probability          *float64 `json:"probability"`
	ProbabilitySource    *string  `json:"probabilitySource"`
	ProbabilityChange24h *float64 `json:"probabilityChange24h"`
}

// EventsRadarResponse is returned by GET /events/radar.
type EventsRadarResponse struct {
	Events    []EventItem `json:"events"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

// ---- Analysis ----

// SubIndicatorDetail carries raw and derived metrics for one sub-indicator.
type SubIndicatorDetail struct {
	Name              string  `json:"name"`
	RawValue          float64 `json:"rawValue"`
	Percentile1y      float64 `json:"percentile1y"`
	Percentile5y      float64 `json:"percentile5y"`
	BlendedPercentile float64 `json:"blendedPercentile"`
	NormalizedScore   float64 `json:"normalizedScore"`
	WeightInDimension float64 `json:"weightInDimension"`
	DataSource        string  `json:"dataSource"`
	DataAsOf          string  `json:"dataAsOf"`
}

// DimensionDetail contains a scoring dimension with its sub-indicators.
type DimensionDetail struct {
	Dimension      string               `json:"dimension"`
	NameZh         string               `json:"nameZh"`
	NameEn         string               `json:"nameEn"`
	Score          float64              `json:"score"`
	BaseScore      float64              `json:"baseScore"`
	LLMAdjustment  float64              `json:"llmAdjustment"`
	LLMAnomalyFlag bool                 `json:"llmAnomalyFlag"`
	Weight         float64              `json:"weight"`
	SubIndicators  []SubIndicatorDetail `json:"subIndicators"`
}

// DrawdownReference holds drawdown benchmark data.
type DrawdownReference struct {
	CurrentBullRunStart   string  `json:"currentBullRunStart"`
	MaxDrawdown           float64 `json:"maxDrawdown"`
	MaxDrawdownDate       string  `json:"maxDrawdownDate"`
	HistoricalAvgDrawdown float64 `json:"historicalAvgDrawdown"`
}

// Scenario describes a conditional execution action within an execution plan.
type Scenario struct {
	Condition      string  `json:"condition"`
	Action         string  `json:"action"`
	LotCount       float64 `json:"lotCount"`
	Rationale      string  `json:"rationale"`
	Priority       int     `json:"priority"`
	ExclusionGroup *string `json:"exclusionGroup,omitempty"`
}

// ExecutionPlanData carries the actionable trade plan for a holding.
type ExecutionPlanData struct {
	Action               string     `json:"action"`
	ActionLabel          string     `json:"actionLabel"`
	DefaultAction        string     `json:"defaultAction"`
	CurrentPosition      float64    `json:"currentPosition"`
	TargetPosition       float64    `json:"targetPosition"`
	Scenarios            []Scenario `json:"scenarios"`
	StopLoss             *float64   `json:"stopLoss,omitempty"`
	TakeProfit           *float64   `json:"takeProfit,omitempty"`
	ValidDays            int        `json:"validDays"`
	NoTriggerNote        *string    `json:"noTriggerNote,omitempty"`
	ConcentrationLevel   string     `json:"concentrationLevel"`
	ConcentrationMessage string     `json:"concentrationMessage"`
	IsDemoPlan           bool       `json:"isDemoPlan"`
}

// AnalysisDetail is the full asset analysis result.
type AnalysisDetail struct {
	AssetAnalysisID      string             `json:"assetAnalysisId"`
	OverallScore         float64            `json:"overallScore"`
	SignalLevel          string             `json:"signalLevel"`
	Confidence           float64            `json:"confidence"`
	ConfidenceBandLow    float64            `json:"confidenceBandLow"`
	ConfidenceBandHigh   float64            `json:"confidenceBandHigh"`
	ModelVersion         string             `json:"modelVersion"`
	MarketInterpretation string             `json:"marketInterpretation"`
	RiskFactors          []string           `json:"riskFactors"`
	RegimeSummary        string             `json:"regimeSummary"`
	ConflictType         *string            `json:"conflictType,omitempty"`
	ConflictMessage      *string            `json:"conflictMessage,omitempty"`
	ScoreDelta           *float64           `json:"scoreDelta,omitempty"`
	ChangeSummary        *string            `json:"changeSummary,omitempty"`
	MajorChangeRecap     *string            `json:"majorChangeRecap,omitempty"`
	USDExchangeRate      *float64           `json:"usdExchangeRate,omitempty"`
	PriceAtAnalysis      *float64           `json:"priceAtAnalysis,omitempty"`
	AnalyzedAt           time.Time          `json:"analyzedAt"`
	GeneratedBy          string             `json:"generatedBy"`
	LLMSkipped           bool               `json:"llmSkipped"`
	DrawdownReference    *DrawdownReference `json:"drawdownReference,omitempty"`
	DemoPlan             *ExecutionPlanData `json:"demoPlan,omitempty"`
	Dimensions           []DimensionDetail  `json:"dimensions"`
}

// HoldingInput carries a user's holding position details for analysis.
type HoldingInput struct {
	HoldingID     int64   `json:"holdingId"`
	CostPrice     string  `json:"costPrice"`
	PositionRatio float64 `json:"positionRatio"`
	Quantity      float64 `json:"quantity"`
}

// AnalyzeHoldingRequest is sent to POST /analyze/holding.
type AnalyzeHoldingRequest struct {
	AssetCode       string       `json:"assetCode"`
	AssetAnalysisID int64        `json:"assetAnalysisId"`
	Holding         HoldingInput `json:"holding"`
	RiskPreference  string       `json:"riskPreference"`
	PeerExposure    float64      `json:"peerExposure"`
	Language        string       `json:"language"`
	LLMConfig       *LLMConfig   `json:"llmConfig,omitempty"`
	RequestID       string       `json:"requestId"`
}

// HoldingAnalysisResponse is returned by POST /analyze/holding.
type HoldingAnalysisResponse struct {
	ExecutionPlan ExecutionPlanData `json:"executionPlan"`
}

// DemoPlanRequest is sent to POST /analyze/demo-plan.
type DemoPlanRequest struct {
	AssetCode string     `json:"assetCode"`
	Language  string     `json:"language"`
	LLMConfig *LLMConfig `json:"llmConfig,omitempty"`
	RequestID string     `json:"requestId"`
}

// DemoPlanResponse is returned by POST /analyze/demo-plan.
type DemoPlanResponse struct {
	ExecutionPlan ExecutionPlanData `json:"executionPlan"`
}

// ---- Content ----

// WeeklyInsightSection is a single section within a weekly insight.
type WeeklyInsightSection struct {
	Heading string `json:"heading"`
	Content string `json:"content"`
}

// WeeklyInsightRequest is sent to POST /content/weekly-insight.
type WeeklyInsightRequest struct {
	Locale    string     `json:"locale"`
	LLMConfig *LLMConfig `json:"llmConfig,omitempty"`
	RequestID string     `json:"requestId"`
}

// WeeklyInsightResponse is returned by POST /content/weekly-insight.
type WeeklyInsightResponse struct {
	Title       string                 `json:"title"`
	Sections    []WeeklyInsightSection `json:"sections"`
	GeneratedAt time.Time              `json:"generatedAt"`
}

// ---- Request types for async job triggers ----

// TriggerAssetAnalysisRequest is sent to POST /jobs/analyze-asset.
type TriggerAssetAnalysisRequest struct {
	AssetCode string     `json:"assetCode"`
	Locale    string     `json:"locale"`
	LLMConfig *LLMConfig `json:"llmConfig,omitempty"`
	RequestID string     `json:"requestId,omitempty"`
}

// BatchAnalyzeAsset is a single entry within a batch analyze request.
type BatchAnalyzeAsset struct {
	AssetCode string `json:"assetCode"`
	Locale    string `json:"locale"`
}

// TriggerBatchAnalysisRequest is sent to POST /jobs/batch-analyze.
type TriggerBatchAnalysisRequest struct {
	Assets    []BatchAnalyzeAsset `json:"assets"`
	LLMConfig *LLMConfig          `json:"llmConfig,omitempty"`
	RequestID string              `json:"requestId,omitempty"`
}

// ---- Health ----

// HealthResponse is returned by GET /health.
type HealthResponse struct {
	Status  string         `json:"status"`
	Checks  map[string]any `json:"checks"`
	Version string         `json:"version"`
	Uptime  int64          `json:"uptime"`
}
