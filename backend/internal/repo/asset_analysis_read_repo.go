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

// AssetAnalysisReadRepo provides read-only access to rs_asset_analyses.
// richman only reads from this table; richson is the sole writer.
type AssetAnalysisReadRepo struct {
	pool *pgxpool.Pool
}

// NewAssetAnalysisReadRepo creates a new AssetAnalysisReadRepo.
func NewAssetAnalysisReadRepo(pool *pgxpool.Pool) *AssetAnalysisReadRepo {
	return &AssetAnalysisReadRepo{pool: pool}
}

// assetAnalysisColumns lists every column scanned by scanAssetAnalysisRow.
// Keep in sync with the scan order in scanAssetAnalysisRow.
const assetAnalysisColumns = `asset_analysis_id, asset_code, locale,
	overall_score, signal_level, confidence, confidence_band_low, confidence_band_high,
	model_version, market_interpretation, risk_factors, regime_summary,
	d1_score, d1_base_score, d1_llm_adjustment,
	d2_score, d2_base_score, d2_llm_adjustment,
	d3_score, d3_base_score, d3_llm_adjustment,
	d4_score, d4_base_score, d4_llm_adjustment,
	d1_weight, d2_weight, d3_weight, d4_weight,
	llm_skipped, data_coverage,
	conflict_type, conflict_message,
	prev_analysis_id, score_delta, change_summary, major_change_recap,
	data_snapshot_at, usd_exchange_rate, price_at_analysis,
	demo_plan, analysis_metadata,
	generated_by, source, job_id,
	analyzed_at, created_at, updated_at, is_deleted`

// analysisScanner abstracts pgx.Row and pgx.Rows for shared scanning logic.
type analysisScanner interface {
	Scan(dest ...any) error
}

// scanAssetAnalysisRow reads the canonical rs_asset_analyses columns into a
// model.AssetAnalysis. The raw byte slices for JSONB columns are decoded after
// scanning to avoid pgx type mapping issues.
func scanAssetAnalysisRow(row analysisScanner, a *model.AssetAnalysis) error {
	var (
		riskFactorsRaw  []byte
		demoPlanRaw     []byte
		analysisMetaRaw []byte
	)
	if err := row.Scan(
		&a.AssetAnalysisID, &a.AssetCode, &a.Locale,
		&a.OverallScore, &a.SignalLevel, &a.Confidence, &a.ConfidenceBandLow, &a.ConfidenceBandHigh,
		&a.ModelVersion, &a.MarketInterpretation, &riskFactorsRaw, &a.RegimeSummary,
		&a.D1Score, &a.D1BaseScore, &a.D1LLMAdjustment,
		&a.D2Score, &a.D2BaseScore, &a.D2LLMAdjustment,
		&a.D3Score, &a.D3BaseScore, &a.D3LLMAdjustment,
		&a.D4Score, &a.D4BaseScore, &a.D4LLMAdjustment,
		&a.D1Weight, &a.D2Weight, &a.D3Weight, &a.D4Weight,
		&a.LLMSkipped, &a.DataCoverage,
		&a.ConflictType, &a.ConflictMessage,
		&a.PrevAnalysisID, &a.ScoreDelta, &a.ChangeSummary, &a.MajorChangeRecap,
		&a.DataSnapshotAt, &a.UsdExchangeRate, &a.PriceAtAnalysis,
		&demoPlanRaw, &analysisMetaRaw,
		&a.GeneratedBy, &a.Source, &a.JobID,
		&a.AnalyzedAt, &a.CreatedAt, &a.UpdatedAt, &a.IsDeleted,
	); err != nil {
		return err
	}
	a.RiskFactors = json.RawMessage(riskFactorsRaw)
	a.DemoPlan = json.RawMessage(demoPlanRaw)
	a.AnalysisMetadata = json.RawMessage(analysisMetaRaw)
	return nil
}

// GetLatestByAssetCode returns the most recent analysis for an asset (highest
// analyzed_at timestamp). Returns nil if no analysis exists.
func (r *AssetAnalysisReadRepo) GetLatestByAssetCode(
	ctx context.Context, code string,
) (*model.AssetAnalysis, error) {
	var a model.AssetAnalysis
	row := r.pool.QueryRow(ctx,
		`SELECT `+assetAnalysisColumns+`
		 FROM rs_asset_analyses
		 WHERE asset_code = $1 AND is_deleted = 0
		 ORDER BY analyzed_at DESC
		 LIMIT 1`,
		code,
	)
	if err := scanAssetAnalysisRow(row, &a); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest analysis by asset code: %w", err)
	}
	return &a, nil
}

// GetSecondLatestByAssetCode returns the second most recent analysis for an
// asset (i.e. the analysis immediately before the one with excludeID). Used by
// the daily briefing to compute the gold score delta vs yesterday.
// Returns nil if no prior analysis exists.
func (r *AssetAnalysisReadRepo) GetSecondLatestByAssetCode(
	ctx context.Context, code string, excludeID int64,
) (*model.AssetAnalysis, error) {
	var a model.AssetAnalysis
	row := r.pool.QueryRow(ctx,
		`SELECT `+assetAnalysisColumns+`
		 FROM rs_asset_analyses
		 WHERE asset_code = $1 AND asset_analysis_id != $2 AND is_deleted = 0
		 ORDER BY analyzed_at DESC
		 LIMIT 1`,
		code, excludeID,
	)
	if err := scanAssetAnalysisRow(row, &a); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query second latest analysis by asset code: %w", err)
	}
	return &a, nil
}

// GetLatestByAssetCodes returns the most recent analysis for each asset in the
// provided codes slice. The result is keyed by asset_code. Codes with no
// analysis are absent from the returned map. An empty codes slice returns an
// empty map without querying the database.
func (r *AssetAnalysisReadRepo) GetLatestByAssetCodes(
	ctx context.Context, codes []string,
) (map[string]*model.AssetAnalysis, error) {
	if len(codes) == 0 {
		return map[string]*model.AssetAnalysis{}, nil
	}

	// Use DISTINCT ON to get the latest row per asset_code in one query.
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (asset_code) `+assetAnalysisColumns+`
		 FROM rs_asset_analyses
		 WHERE asset_code = ANY($1) AND is_deleted = 0
		 ORDER BY asset_code, analyzed_at DESC`,
		codes,
	)
	if err != nil {
		return nil, fmt.Errorf("query latest analyses by asset codes: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*model.AssetAnalysis, len(codes))
	for rows.Next() {
		var a model.AssetAnalysis
		if err := scanAssetAnalysisRow(rows, &a); err != nil {
			return nil, fmt.Errorf("scan asset analysis: %w", err)
		}
		cp := a
		result[a.AssetCode] = &cp
	}
	return result, nil
}

// GetScoresForPercentile returns the overall_score values for an asset over
// the past N days, ordered oldest first. Used by the service layer to compute
// percentile labels (e.g. "近一年偏高").
func (r *AssetAnalysisReadRepo) GetScoresForPercentile(
	ctx context.Context, code string, days int,
) ([]float64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT overall_score
		 FROM rs_asset_analyses
		 WHERE asset_code = $1
		   AND is_deleted = 0
		   AND analyzed_at >= NOW() - ($2 || ' days')::INTERVAL
		 ORDER BY analyzed_at ASC`,
		code, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query scores for percentile: %w", err)
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var s float64
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan score: %w", err)
		}
		scores = append(scores, s)
	}
	return scores, nil
}

// GetSparklineScores returns the N most recent overall_score values for an
// asset, ordered oldest first. Used by the frontend to render sparkline charts.
func (r *AssetAnalysisReadRepo) GetSparklineScores(
	ctx context.Context, code string, limit int,
) ([]float64, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.pool.Query(ctx,
		`SELECT overall_score FROM (
		   SELECT overall_score, analyzed_at
		   FROM rs_asset_analyses
		   WHERE asset_code = $1 AND is_deleted = 0
		   ORDER BY analyzed_at DESC
		   LIMIT $2
		 ) sub
		 ORDER BY analyzed_at ASC`,
		code, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query sparkline scores: %w", err)
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var s float64
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan sparkline score: %w", err)
		}
		scores = append(scores, s)
	}
	return scores, nil
}
