package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// AnalysisJobReadRepo provides read-only access to rs_analysis_jobs.
// richman only reads from this table; richson is the sole writer.
type AnalysisJobReadRepo struct {
	pool *pgxpool.Pool
}

// NewAnalysisJobReadRepo creates a new AnalysisJobReadRepo.
func NewAnalysisJobReadRepo(pool *pgxpool.Pool) *AnalysisJobReadRepo {
	return &AnalysisJobReadRepo{pool: pool}
}

// GetByJobID returns a job by its UUID string. Returns nil if not found.
func (r *AnalysisJobReadRepo) GetByJobID(
	ctx context.Context, jobID string,
) (*model.AnalysisJob, error) {
	var j model.AnalysisJob
	var stepsRaw []byte
	err := r.pool.QueryRow(ctx,
		`SELECT job_id, asset_code, job_type, status, progress,
		        current_step, steps, error_message, error_code,
		        asset_analysis_id, expires_at, started_at, completed_at,
		        request_id, locale, created_at
		 FROM rs_analysis_jobs
		 WHERE job_id = $1`,
		jobID,
	).Scan(
		&j.JobID, &j.AssetCode, &j.JobType, &j.Status, &j.Progress,
		&j.CurrentStep, &stepsRaw, &j.ErrorMessage, &j.ErrorCode,
		&j.AssetAnalysisID, &j.ExpiresAt, &j.StartedAt, &j.CompletedAt,
		&j.RequestID, &j.Locale, &j.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query analysis job by id: %w", err)
	}
	j.Steps = json.RawMessage(stepsRaw)
	return &j, nil
}
