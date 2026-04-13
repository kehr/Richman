package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// InviteRewardRepo handles data access for rm_invite_rewards.
type InviteRewardRepo struct {
	pool *pgxpool.Pool
}

// NewInviteRewardRepo creates a new InviteRewardRepo.
func NewInviteRewardRepo(pool *pgxpool.Pool) *InviteRewardRepo {
	return &InviteRewardRepo{pool: pool}
}

// Create inserts a new reward record and returns the generated reward ID.
// rewardDetail is optional and may be passed as nil.
func (r *InviteRewardRepo) Create(
	ctx context.Context,
	userID int64,
	rewardType string,
	rewardDetail json.RawMessage,
	sourceInviteID int64,
	creator string,
) (int64, error) {
	var rewardID int64
	var detailArg any
	if len(rewardDetail) > 0 {
		detailArg = []byte(rewardDetail)
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO rm_invite_rewards (user_id, reward_type, reward_detail, source_invite_id, creator, modifier)
		 VALUES ($1, $2, $3, $4, $5, $5)
		 RETURNING reward_id`,
		userID, rewardType, detailArg, sourceInviteID, creator,
	).Scan(&rewardID)
	if err != nil {
		return 0, fmt.Errorf("insert invite reward: %w", err)
	}
	return rewardID, nil
}

// ListByUser returns all active (is_deleted = 0) rewards for a user,
// ordered by created_at ascending.
func (r *InviteRewardRepo) ListByUser(
	ctx context.Context, userID int64,
) ([]model.InviteReward, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT reward_id, user_id, reward_type, reward_detail, source_invite_id,
		        created_at, updated_at
		 FROM rm_invite_rewards
		 WHERE user_id = $1 AND is_deleted = 0
		 ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query invite rewards by user: %w", err)
	}
	defer rows.Close()

	var rewards []model.InviteReward
	for rows.Next() {
		var rw model.InviteReward
		var detailRaw []byte
		if err := rows.Scan(
			&rw.RewardID, &rw.UserID, &rw.RewardType, &detailRaw, &rw.SourceInviteID,
			&rw.CreatedAt, &rw.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invite reward: %w", err)
		}
		rw.RewardDetail = json.RawMessage(detailRaw)
		rewards = append(rewards, rw)
	}
	return rewards, nil
}

// CountByTypeAndUser counts how many active rewards of a given type a user has.
// Used by the analysis handler to check extra_analysis_refresh quota.
func (r *InviteRewardRepo) CountByTypeAndUser(
	ctx context.Context, userID int64, rewardType string,
) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM rm_invite_rewards
		 WHERE user_id = $1 AND reward_type = $2 AND is_deleted = 0`,
		userID, rewardType,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count invite rewards: %w", err)
	}
	return count, nil
}
