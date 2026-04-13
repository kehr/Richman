package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// PlanRepo handles plan data access operations.
type PlanRepo struct {
	pool *pgxpool.Pool
}

// NewPlanRepo creates a new PlanRepo.
func NewPlanRepo(pool *pgxpool.Pool) *PlanRepo {
	return &PlanRepo{pool: pool}
}

// GetPlanByID finds a plan by its ID. Returns nil if not found.
func (r *PlanRepo) GetPlanByID(ctx context.Context, planID int64) (*model.Plan, error) {
	var p model.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT plan_id, name, max_holdings, max_daily_analysis, max_push_channels, created_at, updated_at
		 FROM rm_plans
		 WHERE plan_id = $1 AND is_deleted = 0`,
		planID,
	).Scan(&p.PlanID, &p.Name, &p.MaxHoldings, &p.MaxDailyAnalysis, &p.MaxPushChannels, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query plan by id: %w", err)
	}
	return &p, nil
}

// GetPlanByName finds a plan by name. Returns nil if not found.
func (r *PlanRepo) GetPlanByName(ctx context.Context, name string) (*model.Plan, error) {
	var p model.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT plan_id, name, max_holdings, max_daily_analysis, max_push_channels, created_at, updated_at
		 FROM rm_plans
		 WHERE name = $1 AND is_deleted = 0`,
		name,
	).Scan(&p.PlanID, &p.Name, &p.MaxHoldings, &p.MaxDailyAnalysis, &p.MaxPushChannels, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query plan by name: %w", err)
	}
	return &p, nil
}
