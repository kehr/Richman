package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// AnalysisDimensionReadRepo provides read-only access to rs_asset_analysis_dimensions.
// richman only reads from this table; richson is the sole writer.
type AnalysisDimensionReadRepo struct {
	pool *pgxpool.Pool
}

// NewAnalysisDimensionReadRepo creates a new AnalysisDimensionReadRepo.
func NewAnalysisDimensionReadRepo(pool *pgxpool.Pool) *AnalysisDimensionReadRepo {
	return &AnalysisDimensionReadRepo{pool: pool}
}

// GetByAnalysisID returns all dimension records for a given analysis, ordered
// by dimension and sub_indicator for consistent presentation.
func (r *AnalysisDimensionReadRepo) GetByAnalysisID(
	ctx context.Context, analysisID int64,
) ([]model.AnalysisDimension, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, asset_analysis_id, dimension, sub_indicator,
		        raw_value, percentile_1y, percentile_5y, blended_percentile,
		        normalized_score, weight_in_dimension, data_source, data_as_of
		 FROM rs_asset_analysis_dimensions
		 WHERE asset_analysis_id = $1
		 ORDER BY dimension, sub_indicator`,
		analysisID,
	)
	if err != nil {
		return nil, fmt.Errorf("query analysis dimensions: %w", err)
	}
	defer rows.Close()

	var dims []model.AnalysisDimension
	for rows.Next() {
		var d model.AnalysisDimension
		if err := rows.Scan(
			&d.ID, &d.AssetAnalysisID, &d.Dimension, &d.SubIndicator,
			&d.RawValue, &d.Percentile1Y, &d.Percentile5Y, &d.BlendedPercentile,
			&d.NormalizedScore, &d.WeightInDimension, &d.DataSource, &d.DataAsOf,
		); err != nil {
			return nil, fmt.Errorf("scan analysis dimension: %w", err)
		}
		dims = append(dims, d)
	}
	return dims, nil
}
