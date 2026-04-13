package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// EventAlertReadRepo provides read access to rs_event_alerts and allows
// richman to mark alerts as processed via MarkAlerted. This write exception
// is documented in richman-backend-v2-trd.md SS6.1 -- richman updates only
// the alerted flag to avoid richson needing awareness of notification delivery.
type EventAlertReadRepo struct {
	pool *pgxpool.Pool
}

// NewEventAlertReadRepo creates a new EventAlertReadRepo.
func NewEventAlertReadRepo(pool *pgxpool.Pool) *EventAlertReadRepo {
	return &EventAlertReadRepo{pool: pool}
}

// GetUnalerted returns all event alerts that have not yet been processed
// (alerted = false), ordered by detected_at ascending so older alerts are
// handled first.
func (r *EventAlertReadRepo) GetUnalerted(ctx context.Context) ([]model.EventAlert, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, event_slug, event_title, source,
		        prev_probability, curr_probability, delta, threshold,
		        gold_direction, alerted, detected_at
		 FROM rs_event_alerts
		 WHERE alerted = false
		 ORDER BY detected_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query unalerted event alerts: %w", err)
	}
	defer rows.Close()

	var alerts []model.EventAlert
	for rows.Next() {
		var a model.EventAlert
		if err := rows.Scan(
			&a.ID, &a.EventSlug, &a.EventTitle, &a.Source,
			&a.PrevProbability, &a.CurrProbability, &a.Delta, &a.Threshold,
			&a.GoldDirection, &a.Alerted, &a.DetectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

// MarkAlerted marks the specified event alerts as alerted = true. This is the
// sole cross-service write that richman makes to an rs_* table. Production DB
// user richman_user must have UPDATE permission on rs_event_alerts.alerted.
// Empty ids slice is a no-op.
func (r *EventAlertReadRepo) MarkAlerted(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE rs_event_alerts
		 SET alerted = true
		 WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return fmt.Errorf("mark alerts as alerted: %w", err)
	}
	return nil
}
