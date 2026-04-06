package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// InviteRepo handles invite code data access operations.
type InviteRepo struct {
	pool *pgxpool.Pool
}

// NewInviteRepo creates a new InviteRepo.
func NewInviteRepo(pool *pgxpool.Pool) *InviteRepo {
	return &InviteRepo{pool: pool}
}

// GetInviteCodeByCode finds an invite code by its code string. Returns nil if not found.
func (r *InviteRepo) GetInviteCodeByCode(ctx context.Context, code string) (*model.InviteCode, error) {
	var ic model.InviteCode
	err := r.pool.QueryRow(ctx,
		`SELECT invite_code_id, code, max_uses, used_count, created_at, updated_at
		 FROM invite_codes
		 WHERE code = $1 AND is_deleted = 0`,
		code,
	).Scan(&ic.InviteCodeID, &ic.Code, &ic.MaxUses, &ic.UsedCount, &ic.CreatedAt, &ic.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query invite code: %w", err)
	}
	return &ic, nil
}

// IncrementInviteCodeUsage atomically increments the used_count of an invite code.
func (r *InviteRepo) IncrementInviteCodeUsage(ctx context.Context, inviteCodeID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE invite_codes
		 SET used_count = used_count + 1, updated_at = NOW()
		 WHERE invite_code_id = $1 AND is_deleted = 0`,
		inviteCodeID,
	)
	if err != nil {
		return fmt.Errorf("increment invite code usage: %w", err)
	}
	return nil
}
