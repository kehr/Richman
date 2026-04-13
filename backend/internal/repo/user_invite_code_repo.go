package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// UserInviteCodeRepo handles data access for rm_user_invite_codes.
type UserInviteCodeRepo struct {
	pool *pgxpool.Pool
}

// NewUserInviteCodeRepo creates a new UserInviteCodeRepo.
func NewUserInviteCodeRepo(pool *pgxpool.Pool) *UserInviteCodeRepo {
	return &UserInviteCodeRepo{pool: pool}
}

const userInviteCodeColumns = `invite_code_id, user_id, code, is_used,
	used_by_user_id, used_at, created_at, updated_at`

// scanUserInviteCode reads invite code columns into a model.UserInviteCode.
func scanUserInviteCode(row pgx.Row, c *model.UserInviteCode) error {
	return row.Scan(
		&c.InviteCodeID, &c.UserID, &c.Code, &c.IsUsed,
		&c.UsedByUserID, &c.UsedAt, &c.CreatedAt, &c.UpdatedAt,
	)
}

// Create inserts a new invite code for a user and returns the created record.
func (r *UserInviteCodeRepo) Create(
	ctx context.Context, userID int64, code, creator string,
) (*model.UserInviteCode, error) {
	var c model.UserInviteCode
	row := r.pool.QueryRow(ctx,
		`INSERT INTO rm_user_invite_codes (user_id, code, creator, modifier)
		 VALUES ($1, $2, $3, $3)
		 RETURNING `+userInviteCodeColumns,
		userID, code, creator,
	)
	if err := scanUserInviteCode(row, &c); err != nil {
		return nil, fmt.Errorf("insert user invite code: %w", err)
	}
	return &c, nil
}

// CreateWithTx inserts a new invite code for a user inside an existing
// transaction. Semantics are identical to Create.
func (r *UserInviteCodeRepo) CreateWithTx(
	ctx context.Context, tx pgx.Tx, userID int64, code, creator string,
) (*model.UserInviteCode, error) {
	var c model.UserInviteCode
	row := tx.QueryRow(ctx,
		`INSERT INTO rm_user_invite_codes (user_id, code, creator, modifier)
		 VALUES ($1, $2, $3, $3)
		 RETURNING `+userInviteCodeColumns,
		userID, code, creator,
	)
	if err := scanUserInviteCode(row, &c); err != nil {
		return nil, fmt.Errorf("insert user invite code (tx): %w", err)
	}
	return &c, nil
}

// GetByCode finds an active (is_deleted = 0) invite code by its code string.
// Returns nil if not found.
func (r *UserInviteCodeRepo) GetByCode(
	ctx context.Context, code string,
) (*model.UserInviteCode, error) {
	var c model.UserInviteCode
	row := r.pool.QueryRow(ctx,
		`SELECT `+userInviteCodeColumns+`
		 FROM rm_user_invite_codes
		 WHERE code = $1 AND is_deleted = 0`,
		code,
	)
	if err := scanUserInviteCode(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query invite code by code: %w", err)
	}
	return &c, nil
}

// ListByUser returns all active (is_deleted = 0) invite codes for a user,
// ordered by created_at ascending so codes appear in generation order.
func (r *UserInviteCodeRepo) ListByUser(
	ctx context.Context, userID int64,
) ([]model.UserInviteCode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userInviteCodeColumns+`
		 FROM rm_user_invite_codes
		 WHERE user_id = $1 AND is_deleted = 0
		 ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query invite codes by user: %w", err)
	}
	defer rows.Close()

	var codes []model.UserInviteCode
	for rows.Next() {
		var c model.UserInviteCode
		if err := rows.Scan(
			&c.InviteCodeID, &c.UserID, &c.Code, &c.IsUsed,
			&c.UsedByUserID, &c.UsedAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invite code: %w", err)
		}
		codes = append(codes, c)
	}
	return codes, nil
}

// CountByUser returns the total number of active invite codes for a user.
// Used to enforce the per-user limit of 20 invite codes.
func (r *UserInviteCodeRepo) CountByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM rm_user_invite_codes
		 WHERE user_id = $1 AND is_deleted = 0`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count invite codes by user: %w", err)
	}
	return count, nil
}

// CountUnusedByUser returns the number of unused (is_used = false) active
// invite codes for a user. Used to decide whether to generate a new code on
// login streak milestones.
func (r *UserInviteCodeRepo) CountUnusedByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM rm_user_invite_codes
		 WHERE user_id = $1 AND is_used = false AND is_deleted = 0`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unused invite codes by user: %w", err)
	}
	return count, nil
}

// ConsumeCode atomically marks an invite code as used. Uses a conditional
// UPDATE with a RETURNING clause to detect concurrent consumption: if another
// request consumed the code between lookup and this call, RETURNING returns no
// rows. Returns the updated record, or nil if the code was already consumed.
func (r *UserInviteCodeRepo) ConsumeCode(
	ctx context.Context, tx pgx.Tx, inviteCodeID, newUserID int64,
) (*model.UserInviteCode, error) {
	var c model.UserInviteCode
	row := tx.QueryRow(ctx,
		`UPDATE rm_user_invite_codes
		 SET is_used = true,
		     used_by_user_id = $2,
		     used_at = NOW(),
		     updated_at = NOW()
		 WHERE invite_code_id = $1 AND is_used = false AND is_deleted = 0
		 RETURNING `+userInviteCodeColumns,
		inviteCodeID, newUserID,
	)
	if err := scanUserInviteCode(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // already consumed by concurrent request
		}
		return nil, fmt.Errorf("consume invite code: %w", err)
	}
	return &c, nil
}

// GetFirstAvailable returns the first unused (is_used = false) active invite
// code for a user. Used by the share link endpoint to embed an invite code in
// the share URL. Returns nil if no unused code exists.
func (r *UserInviteCodeRepo) GetFirstAvailable(
	ctx context.Context, userID int64,
) (*model.UserInviteCode, error) {
	var c model.UserInviteCode
	row := r.pool.QueryRow(ctx,
		`SELECT `+userInviteCodeColumns+`
		 FROM rm_user_invite_codes
		 WHERE user_id = $1 AND is_used = false AND is_deleted = 0
		 ORDER BY created_at ASC
		 LIMIT 1`,
		userID,
	)
	if err := scanUserInviteCode(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query first available invite code: %w", err)
	}
	return &c, nil
}

// ClearUsedByForUser nullifies used_by_user_id on all invite codes that were
// consumed by the given user. Called during account deletion so that the
// invite codes are no longer linked to the soft-deleted user record.
func (r *UserInviteCodeRepo) ClearUsedByForUser(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE rm_user_invite_codes
		 SET used_by_user_id = NULL,
		     updated_at      = NOW()
		 WHERE used_by_user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("clear used_by_user_id for user %d: %w", userID, err)
	}
	return nil
}

// ListInvitedUsers returns basic info about users who were invited by the
// given user (i.e. used_by_user_id records joined to rm_users for display).
// The email is masked via model.MaskName before being set on InvitedUserName.
func (r *UserInviteCodeRepo) ListInvitedUsers(
	ctx context.Context, userID int64,
) ([]model.InvitedUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT u.user_id, u.email, c.used_at
		 FROM rm_user_invite_codes c
		 JOIN rm_users u ON u.user_id = c.used_by_user_id
		 WHERE c.user_id = $1
		   AND c.is_used = true
		   AND c.is_deleted = 0
		   AND u.is_deleted = 0
		 ORDER BY c.used_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query invited users: %w", err)
	}
	defer rows.Close()

	var users []model.InvitedUser
	for rows.Next() {
		var (
			invitedUserID int64
			email         string
			usedAt        *time.Time
		)
		if err := rows.Scan(&invitedUserID, &email, &usedAt); err != nil {
			return nil, fmt.Errorf("scan invited user: %w", err)
		}
		iu := model.InvitedUser{
			InvitedUserID:   invitedUserID,
			InvitedUserName: model.MaskName(email),
		}
		if usedAt != nil {
			iu.InvitedAt = *usedAt
		}
		users = append(users, iu)
	}
	return users, nil
}
