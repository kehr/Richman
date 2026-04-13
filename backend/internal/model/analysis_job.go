package model

import (
	"encoding/json"
	"time"
)

// AnalysisJob maps to rs_analysis_jobs (read-only from richman).
// Jobs are created and updated by richson; richman reads status for the
// GET /api/v2/analysis/jobs/:jobId endpoint.
type AnalysisJob struct {
	JobID           string
	AssetCode       string
	JobType         string
	Status          string
	Progress        float64
	CurrentStep     *string
	Steps           json.RawMessage
	ErrorMessage    *string
	ErrorCode       *string
	AssetAnalysisID *int64
	ExpiresAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	RequestID       *string
	Locale          *string
	CreatedAt       time.Time
}
