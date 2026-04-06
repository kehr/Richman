package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// AnalysisResultRepo handles analysis result data access operations.
type AnalysisResultRepo struct {
	pool *pgxpool.Pool
}

// NewAnalysisResultRepo creates a new AnalysisResultRepo.
func NewAnalysisResultRepo(pool *pgxpool.Pool) *AnalysisResultRepo {
	return &AnalysisResultRepo{pool: pool}
}

// CreateAnalysisResult inserts a new analysis result record.
func (r *AnalysisResultRepo) CreateAnalysisResult(
	ctx context.Context, userID, holdingID int64, assetCode, rawData string,
) (*model.AnalysisResultRecord, error) {
	var rec model.AnalysisResultRecord
	err := r.pool.QueryRow(ctx,
		`INSERT INTO analysis_results (user_id, holding_id, asset_code, raw_data)
		 VALUES ($1, $2, $3, $4)
		 RETURNING analysis_result_id, user_id, holding_id, asset_code, raw_data, created_at`,
		userID, holdingID, assetCode, rawData,
	).Scan(&rec.ResultID, &rec.UserID, &rec.HoldingID, &rec.AssetCode, &rec.RawData, &rec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert analysis result: %w", err)
	}
	return &rec, nil
}
