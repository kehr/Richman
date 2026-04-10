package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// DecisionCardRepo handles decision card data access operations.
type DecisionCardRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewDecisionCardRepo creates a new DecisionCardRepo. The logger is used to
// surface JSONB unmarshal failures (e.g. corrupted recommendation_json rows)
// that would otherwise produce silently zero-valued recommendations.
func NewDecisionCardRepo(pool *pgxpool.Pool, logger *zap.Logger) *DecisionCardRepo {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DecisionCardRepo{pool: pool, logger: logger}
}

// Pool returns the underlying pgx pool. Exposed so the service layer can
// begin transactions that span multiple repos without breaking encapsulation.
func (r *DecisionCardRepo) Pool() *pgxpool.Pool {
	return r.pool
}

// cardColumns enumerates every column selected when scanning a DecisionCard.
// Keep the order in sync with scanCardRow / insertDecisionCardSQL below.
// synthesis_source / provider_used were added by the LLM degraded-contract
// migration (012); they are nullable in the schema so historical rows stay
// readable without a backfill round trip.
const cardColumns = `decision_card_id, user_id, holding_id,
	asset_code, asset_name, asset_type,
	cost_price, current_price, quantity, position_ratio,
	trend_direction, trend_summary,
	position_direction, position_summary,
	catalyst_direction, catalyst_summary,
	confidence,
	action_advice, detailed_advice, risk_warnings,
	today_highlights,
	weight_trend, weight_position, weight_catalyst,
	analyzed_at, created_at,
	recommendation_json, action_level, target_position_ratio,
	badge_state, confidence_delta, prev_card_id, execution_fingerprint,
	synthesis_source, provider_used`

// rowScanner abstracts pgx.Row and pgx.Rows for shared scanning logic.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanCardRow reads a full decision card row into a model.DecisionCard.
// Handles JSONB deserialization for risk_warnings and recommendation_json.
// A failure to decode recommendation_json is logged at warn level so badge
// diffs that depend on the structured recommendation cannot silently degrade
// to zero values.
func (r *DecisionCardRepo) scanCardRow(row rowScanner) (*model.DecisionCard, error) {
	var card model.DecisionCard
	var riskData []byte
	var recData []byte
	err := row.Scan(
		&card.CardID, &card.UserID, &card.HoldingID,
		&card.AssetCode, &card.AssetName, &card.AssetType,
		&card.CostPrice, &card.CurrentPrice, &card.Quantity, &card.PositionRatio,
		&card.TrendDirection, &card.TrendSummary,
		&card.PositionDirection, &card.PositionSummary,
		&card.CatalystDirection, &card.CatalystSummary,
		&card.Confidence,
		&card.ActionAdvice, &card.DetailedAdvice, &riskData,
		&card.TodayHighlights,
		&card.WeightTrend, &card.WeightPosition, &card.WeightCatalyst,
		&card.AnalyzedAt, &card.CreatedAt,
		&recData, &card.ActionLevel, &card.TargetPositionRatio,
		&card.BadgeState, &card.ConfidenceDelta, &card.PrevCardID, &card.ExecutionFingerprint,
		&card.SynthesisSource, &card.ProviderUsed,
	)
	if err != nil {
		return nil, err
	}
	if jsonErr := json.Unmarshal(riskData, &card.RiskWarnings); jsonErr != nil {
		// Corrupted risk_warnings JSON would otherwise silently collapse to an
		// empty list and hide the problem from operators. Log a warning with
		// enough context to correlate and fall through to the empty slice.
		r.logger.Warn("decode risk_warnings failed",
			zap.Int64("card_id", card.CardID),
			zap.Int64("holding_id", card.HoldingID),
			zap.Error(jsonErr),
		)
		card.RiskWarnings = nil
	}
	if len(recData) > 0 {
		if jsonErr := json.Unmarshal(recData, &card.Recommendation); jsonErr != nil {
			r.logger.Warn("decode recommendation_json failed",
				zap.Int64("card_id", card.CardID),
				zap.Int64("holding_id", card.HoldingID),
				zap.Error(jsonErr),
			)
		}
	}
	return &card, nil
}

// insertDecisionCardQuerier is the minimal subset of pgxpool.Pool / pgx.Tx
// needed to execute the decision card INSERT. Both types satisfy it, so the
// same insert logic can run inside or outside a transaction.
type insertDecisionCardQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const insertDecisionCardSQL = `INSERT INTO decision_cards
	 (user_id, holding_id, asset_code, asset_name, asset_type,
	  cost_price, current_price, quantity, position_ratio,
	  trend_direction, trend_summary,
	  position_direction, position_summary,
	  catalyst_direction, catalyst_summary,
	  confidence,
	  action_advice, detailed_advice, risk_warnings,
	  today_highlights,
	  weight_trend, weight_position, weight_catalyst,
	  analyzed_at,
	  recommendation_json, action_level, target_position_ratio,
	  badge_state, confidence_delta, prev_card_id, execution_fingerprint,
	  synthesis_source, provider_used)
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
	         $13, $14, $15, $16, $17, $18, $19, $20, $21, $22,
	         $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33)
	 RETURNING ` + cardColumns

func (r *DecisionCardRepo) insertDecisionCardOn(
	ctx context.Context, q insertDecisionCardQuerier, card *model.DecisionCard,
) (*model.DecisionCard, error) {
	riskJSON, err := card.RiskWarningsJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal risk warnings: %w", err)
	}
	recJSON, err := card.RecommendationJSONBytes()
	if err != nil {
		return nil, fmt.Errorf("marshal recommendation: %w", err)
	}

	row := q.QueryRow(ctx, insertDecisionCardSQL,
		card.UserID, card.HoldingID, card.AssetCode, card.AssetName, card.AssetType,
		card.CostPrice, card.CurrentPrice, card.Quantity, card.PositionRatio,
		card.TrendDirection, card.TrendSummary,
		card.PositionDirection, card.PositionSummary,
		card.CatalystDirection, card.CatalystSummary,
		card.Confidence,
		card.ActionAdvice, card.DetailedAdvice, riskJSON,
		card.TodayHighlights,
		card.WeightTrend, card.WeightPosition, card.WeightCatalyst,
		card.AnalyzedAt,
		recJSON, card.ActionLevel, card.TargetPositionRatio,
		card.BadgeState, card.ConfidenceDelta, card.PrevCardID, card.ExecutionFingerprint,
		card.SynthesisSource, card.ProviderUsed,
	)
	inserted, err := r.scanCardRow(row)
	if err != nil {
		return nil, fmt.Errorf("insert decision card: %w", err)
	}
	return inserted, nil
}

// CreateDecisionCard inserts a new decision card using the pool directly.
// Use CreateDecisionCardTx when the insert must participate in a transaction
// alongside a read of the previous card.
func (r *DecisionCardRepo) CreateDecisionCard(
	ctx context.Context, card *model.DecisionCard,
) (*model.DecisionCard, error) {
	return r.insertDecisionCardOn(ctx, r.pool, card)
}

// CreateDecisionCardTx inserts a decision card inside an existing transaction.
func (r *DecisionCardRepo) CreateDecisionCardTx(
	ctx context.Context, tx pgx.Tx, card *model.DecisionCard,
) (*model.DecisionCard, error) {
	return r.insertDecisionCardOn(ctx, tx, card)
}

// GetLatestByHolding returns the most recent decision card for a holding
// (ordered by analyzed_at desc), or nil if none exists. Does not return an
// error on "no rows" so callers can distinguish first analysis from a real DB
// failure.
func (r *DecisionCardRepo) GetLatestByHolding(
	ctx context.Context, holdingID int64,
) (*model.DecisionCard, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+cardColumns+`
		 FROM decision_cards
		 WHERE holding_id = $1 AND is_deleted = 0
		 ORDER BY analyzed_at DESC
		 LIMIT 1`,
		holdingID,
	)
	card, err := r.scanCardRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest decision card by holding: %w", err)
	}
	return card, nil
}

// GetLatestByHoldingTx is the transactional variant of GetLatestByHolding.
// It uses SELECT ... FOR UPDATE to serialize concurrent analyses writing to
// the same holding, preventing prev_card_id drift.
func (r *DecisionCardRepo) GetLatestByHoldingTx(
	ctx context.Context, tx pgx.Tx, holdingID int64,
) (*model.DecisionCard, error) {
	row := tx.QueryRow(ctx,
		`SELECT `+cardColumns+`
		 FROM decision_cards
		 WHERE holding_id = $1 AND is_deleted = 0
		 ORDER BY analyzed_at DESC
		 LIMIT 1
		 FOR UPDATE`,
		holdingID,
	)
	card, err := r.scanCardRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest decision card by holding tx: %w", err)
	}
	return card, nil
}

// GetByID returns a single decision card by its ID. Returns nil if not found.
func (r *DecisionCardRepo) GetByID(ctx context.Context, cardID int64) (*model.DecisionCard, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+cardColumns+`
		 FROM decision_cards
		 WHERE decision_card_id = $1 AND is_deleted = 0`,
		cardID,
	)
	card, err := r.scanCardRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query decision card by id: %w", err)
	}
	return card, nil
}

// ListLatestByUser returns the most recent decision card for each holding of a user.
func (r *DecisionCardRepo) ListLatestByUser(ctx context.Context, userID int64) ([]model.DecisionCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (holding_id) `+cardColumns+`
		 FROM decision_cards
		 WHERE user_id = $1 AND is_deleted = 0
		 ORDER BY holding_id, created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list latest decision cards: %w", err)
	}
	defer rows.Close()

	var cards []model.DecisionCard
	for rows.Next() {
		card, scanErr := r.scanCardRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan decision card: %w", scanErr)
		}
		cards = append(cards, *card)
	}
	return cards, nil
}

// NeedsReanalysis returns true when the user has at least one active
// decision card whose synthesis_source is "template" or "mixed", meaning
// the card was produced (partially or fully) by the deterministic
// fallback and would benefit from a fresh LLM rerun once a provider is
// healthy. The SQL uses EXISTS so the query short-circuits on the first
// match and does not load every card into memory.
func (r *DecisionCardRepo) NeedsReanalysis(ctx context.Context, userID int64) (bool, error) {
	var needs bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
		   SELECT 1 FROM decision_cards
		   WHERE user_id = $1
		     AND is_deleted = 0
		     AND synthesis_source IN ('template', 'mixed')
		 )`,
		userID,
	).Scan(&needs)
	if err != nil {
		return false, fmt.Errorf("query needs reanalysis: %w", err)
	}
	return needs, nil
}

// ListHistory returns the N most recent decision cards for a user across all holdings.
func (r *DecisionCardRepo) ListHistory(ctx context.Context, userID int64, limit int) ([]model.DecisionCard, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+cardColumns+`
		 FROM decision_cards
		 WHERE user_id = $1 AND is_deleted = 0
		 ORDER BY created_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list decision card history: %w", err)
	}
	defer rows.Close()

	var cards []model.DecisionCard
	for rows.Next() {
		card, scanErr := r.scanCardRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan decision card: %w", scanErr)
		}
		cards = append(cards, *card)
	}
	return cards, nil
}
