package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserFeedbackRepo handles CRUD for rm_user_feedback.
type UserFeedbackRepo struct {
	pool *pgxpool.Pool
}

// NewUserFeedbackRepo creates a new UserFeedbackRepo.
func NewUserFeedbackRepo(pool *pgxpool.Pool) *UserFeedbackRepo {
	return &UserFeedbackRepo{pool: pool}
}

// Create inserts a new feedback record and returns the generated feedback ID.
// rating must be one of the values defined by the CHECK constraint on the table.
// comment is optional (pass empty string for no comment).
func (r *UserFeedbackRepo) Create(
	ctx context.Context,
	userID int64,
	analysisID int64,
	rating string,
	comment string,
) (int64, error) {
	var feedbackID int64
	var commentArg any
	if comment != "" {
		commentArg = comment
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO rm_user_feedback (user_id, asset_analysis_id, rating, comment)
		 VALUES ($1, $2, $3, $4)
		 RETURNING feedback_id`,
		userID, analysisID, rating, commentArg,
	).Scan(&feedbackID)
	if err != nil {
		return 0, fmt.Errorf("insert user feedback: %w", err)
	}
	return feedbackID, nil
}
