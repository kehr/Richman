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

// DecisionCardRepo handles decision card data access operations.
type DecisionCardRepo struct {
	pool *pgxpool.Pool
}

// NewDecisionCardRepo creates a new DecisionCardRepo.
func NewDecisionCardRepo(pool *pgxpool.Pool) *DecisionCardRepo {
	return &DecisionCardRepo{pool: pool}
}

// CreateDecisionCard inserts a new decision card and returns it.
func (r *DecisionCardRepo) CreateDecisionCard(
	ctx context.Context, card *model.DecisionCard,
) (*model.DecisionCard, error) {
	riskJSON, err := card.RiskWarningsJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal risk warnings: %w", err)
	}

	var result model.DecisionCard
	var riskData []byte
	err = r.pool.QueryRow(ctx,
		`INSERT INTO decision_cards
		 (user_id, holding_id, asset_code, asset_name, asset_type,
		  cost_price, position_ratio,
		  trend_direction, trend_summary,
		  position_direction, position_summary,
		  catalyst_direction, catalyst_summary,
		  confidence, recommendation,
		  action_advice, detailed_advice, risk_warnings,
		  today_highlights,
		  weight_trend, weight_position, weight_catalyst,
		  analyzed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
		         $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		 RETURNING decision_card_id, user_id, holding_id, asset_code, asset_name, asset_type,
		           cost_price, position_ratio,
		           trend_direction, trend_summary,
		           position_direction, position_summary,
		           catalyst_direction, catalyst_summary,
		           confidence, recommendation,
		           action_advice, detailed_advice, risk_warnings,
		           today_highlights,
		           weight_trend, weight_position, weight_catalyst,
		           analyzed_at, created_at`,
		card.UserID, card.HoldingID, card.AssetCode, card.AssetName, card.AssetType,
		card.CostPrice, card.PositionRatio,
		card.TrendDirection, card.TrendSummary,
		card.PositionDirection, card.PositionSummary,
		card.CatalystDirection, card.CatalystSummary,
		card.Confidence, card.Recommendation,
		card.ActionAdvice, card.DetailedAdvice, riskJSON,
		card.TodayHighlights,
		card.WeightTrend, card.WeightPosition, card.WeightCatalyst,
		card.AnalyzedAt,
	).Scan(
		&result.CardID, &result.UserID, &result.HoldingID,
		&result.AssetCode, &result.AssetName, &result.AssetType,
		&result.CostPrice, &result.PositionRatio,
		&result.TrendDirection, &result.TrendSummary,
		&result.PositionDirection, &result.PositionSummary,
		&result.CatalystDirection, &result.CatalystSummary,
		&result.Confidence, &result.Recommendation,
		&result.ActionAdvice, &result.DetailedAdvice, &riskData,
		&result.TodayHighlights,
		&result.WeightTrend, &result.WeightPosition, &result.WeightCatalyst,
		&result.AnalyzedAt, &result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert decision card: %w", err)
	}

	if err := json.Unmarshal(riskData, &result.RiskWarnings); err != nil {
		result.RiskWarnings = nil
	}

	return &result, nil
}

// scanCard scans a single decision card row.
func scanCard(row pgx.Row) (*model.DecisionCard, error) {
	var card model.DecisionCard
	var riskData []byte
	err := row.Scan(
		&card.CardID, &card.UserID, &card.HoldingID,
		&card.AssetCode, &card.AssetName, &card.AssetType,
		&card.CostPrice, &card.PositionRatio,
		&card.TrendDirection, &card.TrendSummary,
		&card.PositionDirection, &card.PositionSummary,
		&card.CatalystDirection, &card.CatalystSummary,
		&card.Confidence, &card.Recommendation,
		&card.ActionAdvice, &card.DetailedAdvice, &riskData,
		&card.TodayHighlights,
		&card.WeightTrend, &card.WeightPosition, &card.WeightCatalyst,
		&card.AnalyzedAt, &card.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(riskData, &card.RiskWarnings); err != nil {
		card.RiskWarnings = nil
	}
	return &card, nil
}

const cardColumns = `decision_card_id, user_id, holding_id,
	asset_code, asset_name, asset_type,
	cost_price, position_ratio,
	trend_direction, trend_summary,
	position_direction, position_summary,
	catalyst_direction, catalyst_summary,
	confidence, recommendation,
	action_advice, detailed_advice, risk_warnings,
	today_highlights,
	weight_trend, weight_position, weight_catalyst,
	analyzed_at, created_at`

// GetByID returns a single decision card by its ID. Returns nil if not found.
func (r *DecisionCardRepo) GetByID(ctx context.Context, cardID int64) (*model.DecisionCard, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+cardColumns+`
		 FROM decision_cards
		 WHERE decision_card_id = $1 AND is_deleted = 0`,
		cardID,
	)
	card, err := scanCard(row)
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
		var card model.DecisionCard
		var riskData []byte
		if err := rows.Scan(
			&card.CardID, &card.UserID, &card.HoldingID,
			&card.AssetCode, &card.AssetName, &card.AssetType,
			&card.CostPrice, &card.PositionRatio,
			&card.TrendDirection, &card.TrendSummary,
			&card.PositionDirection, &card.PositionSummary,
			&card.CatalystDirection, &card.CatalystSummary,
			&card.Confidence, &card.Recommendation,
			&card.ActionAdvice, &card.DetailedAdvice, &riskData,
			&card.TodayHighlights,
			&card.WeightTrend, &card.WeightPosition, &card.WeightCatalyst,
			&card.AnalyzedAt, &card.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan decision card: %w", err)
		}
		if err := json.Unmarshal(riskData, &card.RiskWarnings); err != nil {
			card.RiskWarnings = nil
		}
		cards = append(cards, card)
	}
	return cards, nil
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
		var card model.DecisionCard
		var riskData []byte
		if err := rows.Scan(
			&card.CardID, &card.UserID, &card.HoldingID,
			&card.AssetCode, &card.AssetName, &card.AssetType,
			&card.CostPrice, &card.PositionRatio,
			&card.TrendDirection, &card.TrendSummary,
			&card.PositionDirection, &card.PositionSummary,
			&card.CatalystDirection, &card.CatalystSummary,
			&card.Confidence, &card.Recommendation,
			&card.ActionAdvice, &card.DetailedAdvice, &riskData,
			&card.TodayHighlights,
			&card.WeightTrend, &card.WeightPosition, &card.WeightCatalyst,
			&card.AnalyzedAt, &card.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan decision card: %w", err)
		}
		if err := json.Unmarshal(riskData, &card.RiskWarnings); err != nil {
			card.RiskWarnings = nil
		}
		cards = append(cards, card)
	}
	return cards, nil
}
