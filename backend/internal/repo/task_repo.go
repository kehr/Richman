package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// AnalysisTaskRepo persists analysis task status history.
type AnalysisTaskRepo struct {
	pool *pgxpool.Pool
}

// NewAnalysisTaskRepo creates a repo instance.
func NewAnalysisTaskRepo(pool *pgxpool.Pool) *AnalysisTaskRepo {
	return &AnalysisTaskRepo{pool: pool}
}

// Upsert creates or updates a task status.
func (r *AnalysisTaskRepo) Upsert(ctx context.Context, task *model.TaskStatus) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO rm_analysis_tasks (task_id, user_id, status, progress, error, started_at, done_at)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
		 ON CONFLICT (task_id) DO UPDATE SET
		 status = EXCLUDED.status,
		 progress = EXCLUDED.progress,
		 error = EXCLUDED.error,
		 done_at = EXCLUDED.done_at`,
		task.TaskID,
		task.UserID,
		task.Status,
		task.Progress,
		task.Error,
		task.StartedAt,
		task.DoneAt,
	)
	return err
}

// GetByID fetches a task status by ID.
func (r *AnalysisTaskRepo) GetByID(ctx context.Context, taskID string) (*model.TaskStatus, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT task_id, user_id, status, progress, COALESCE(error, ''), started_at, done_at
		 FROM rm_analysis_tasks WHERE task_id = $1`, taskID,
	)
	var task model.TaskStatus
	if err := row.Scan(
		&task.TaskID,
		&task.UserID,
		&task.Status,
		&task.Progress,
		&task.Error,
		&task.StartedAt,
		&task.DoneAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	// DB-restored tasks do not persist Holdings/Steps/Logs; initialize to
	// empty slices so the JSON response always sends [] instead of null.
	task.Holdings = make([]model.HoldingProgress, 0)
	task.Steps = make([]model.TaskStep, 0)
	task.Logs = make([]model.TaskLog, 0)
	return &task, nil
}

// FailOrphaned marks all running/pending tasks as failed. Called on server
// startup to recover tasks whose goroutines were killed by a restart.
// Returns the number of rows updated.
func (r *AnalysisTaskRepo) FailOrphaned(ctx context.Context) (int64, error) {
	now := time.Now()
	tag, err := r.pool.Exec(ctx,
		`UPDATE rm_analysis_tasks
		 SET status = 'failed',
		     error  = 'interrupted: server restarted',
		     done_at = $1
		 WHERE status IN ('running', 'pending')`,
		now,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteOlderThan removes persisted tasks older than the cutoff (only completed/failed).
func (r *AnalysisTaskRepo) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM rm_analysis_tasks WHERE done_at IS NOT NULL AND done_at < $1`, cutoff,
	)
	return err
}
