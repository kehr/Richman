package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
	"github.com/shopspring/decimal"
)

// UserRepo handles user data access operations.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// userSelectColumns is the canonical column list for SELECT queries against
// users so every Get* method stays in sync.
const userSelectColumns = `user_id, email, password_hash, role, plan_id,
	risk_preference, total_capital_cny, onboarding_completed_at, categories,
	created_at, updated_at`

// UserSettingsPatch carries a sparse update to the profile fields managed by
// the user_settings service. A nil field means "leave unchanged". To clear a
// nullable field set the matching Clear* flag.
type UserSettingsPatch struct {
	TotalCapitalCNY      *float64
	ClearTotalCapitalCNY bool
	RiskPreference       *string
	Categories           *[]string
}

// scanUser reads the canonical user columns into a model.User, handling the
// nullable decimal / jsonb shapes returned by Postgres.
func scanUser(row pgx.Row, u *model.User) error {
	var (
		totalCap      decimal.NullDecimal
		onboardedAt   *time.Time
		categoriesRaw []byte
	)
	if err := row.Scan(
		&u.UserID, &u.Email, &u.PasswordHash, &u.Role, &u.PlanID,
		&u.RiskPreference, &totalCap, &onboardedAt, &categoriesRaw,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return err
	}
	if totalCap.Valid {
		f, _ := totalCap.Decimal.Float64()
		u.TotalCapitalCNY = &f
	}
	u.OnboardingCompletedAt = onboardedAt
	if len(categoriesRaw) > 0 {
		if err := json.Unmarshal(categoriesRaw, &u.Categories); err != nil {
			return fmt.Errorf("unmarshal categories: %w", err)
		}
	}
	if u.Categories == nil {
		u.Categories = []string{}
	}
	return nil
}

// CreateUser inserts a new user and returns the created user.
func (r *UserRepo) CreateUser(
	ctx context.Context, email, passwordHash, role string, planID int64,
) (*model.User, error) {
	var u model.User
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role, plan_id, creator, modifier)
		 VALUES ($1, $2, $3, $4, $5, $5)
		 RETURNING `+userSelectColumns,
		email, passwordHash, role, planID, email,
	)
	if err := scanUser(row, &u); err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

// GetUserByEmail finds a user by email address. Returns nil if not found.
func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectColumns+`
		 FROM users
		 WHERE email = $1 AND is_deleted = 0`,
		email,
	)
	if err := scanUser(row, &u); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	return &u, nil
}

// GetUserByID finds a user by their ID. Returns nil if not found.
func (r *UserRepo) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	var u model.User
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectColumns+`
		 FROM users
		 WHERE user_id = $1 AND is_deleted = 0`,
		userID,
	)
	if err := scanUser(row, &u); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return &u, nil
}

// GetRiskPreference fetches only the user's risk_preference. Returns an empty
// string (treated as neutral by callers) if the user does not exist so the
// caller can fall back to the default without error handling ceremony.
func (r *UserRepo) GetRiskPreference(ctx context.Context, userID int64) (string, error) {
	var pref string
	err := r.pool.QueryRow(ctx,
		`SELECT risk_preference
		 FROM users
		 WHERE user_id = $1 AND is_deleted = 0`,
		userID,
	).Scan(&pref)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("query user risk preference: %w", err)
	}
	return pref, nil
}

// UpdateUserSettings applies a sparse patch to the user_settings-managed
// profile columns and returns the updated user row. Fields whose patch value
// is nil are left unchanged. TotalCapitalCNY can be cleared to NULL by setting
// ClearTotalCapitalCNY = true (in which case TotalCapitalCNY is ignored).
func (r *UserRepo) UpdateUserSettings(
	ctx context.Context, userID int64, patch *UserSettingsPatch,
) (*model.User, error) {
	if patch == nil {
		patch = &UserSettingsPatch{}
	}

	// Build total_capital_cny value: explicit nil (cleared) vs skip vs set.
	var (
		totalCapArg any
		clearCap    bool
	)
	switch {
	case patch.ClearTotalCapitalCNY:
		clearCap = true
		totalCapArg = nil
	case patch.TotalCapitalCNY != nil:
		totalCapArg = decimal.NewFromFloat(*patch.TotalCapitalCNY)
	default:
		totalCapArg = nil
	}

	var riskArg any
	if patch.RiskPreference != nil {
		riskArg = *patch.RiskPreference
	}

	var categoriesArg any
	if patch.Categories != nil {
		raw, err := json.Marshal(*patch.Categories)
		if err != nil {
			return nil, fmt.Errorf("marshal categories: %w", err)
		}
		categoriesArg = raw
	}

	// COALESCE preserves existing value when the parameter is NULL. For
	// ClearTotalCapitalCNY we need to force NULL, so we branch the SET clause.
	capExpr := "COALESCE($1::NUMERIC, total_capital_cny)"
	if clearCap {
		capExpr = "NULL"
	}

	query := `UPDATE users
		SET total_capital_cny = ` + capExpr + `,
		    risk_preference   = COALESCE($2::VARCHAR, risk_preference),
		    categories        = COALESCE($3::JSONB, categories),
		    updated_at        = NOW()
		WHERE user_id = $4 AND is_deleted = 0
		RETURNING ` + userSelectColumns

	var u model.User
	row := r.pool.QueryRow(ctx, query, totalCapArg, riskArg, categoriesArg, userID)
	if err := scanUser(row, &u); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update user settings: %w", err)
	}
	return &u, nil
}

// MarkOnboardingCompleted stamps onboarding_completed_at with NOW() if it is
// still NULL and returns the refreshed user.
func (r *UserRepo) MarkOnboardingCompleted(
	ctx context.Context, userID int64,
) (*model.User, error) {
	var u model.User
	row := r.pool.QueryRow(ctx,
		`UPDATE users
		 SET onboarding_completed_at = COALESCE(onboarding_completed_at, NOW()),
		     updated_at = NOW()
		 WHERE user_id = $1 AND is_deleted = 0
		 RETURNING `+userSelectColumns,
		userID,
	)
	if err := scanUser(row, &u); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("mark onboarding completed: %w", err)
	}
	return &u, nil
}
