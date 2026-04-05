package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// UserRepo handles user data access operations.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// CreateUser inserts a new user and returns the created user.
func (r *UserRepo) CreateUser(
	ctx context.Context, email, passwordHash, role string, planID int64,
) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role, plan_id, creator, modifier)
		 VALUES ($1, $2, $3, $4, $5, $5)
		 RETURNING user_id, email, password_hash, role, plan_id, created_at, updated_at`,
		email, passwordHash, role, planID, email,
	).Scan(&u.UserID, &u.Email, &u.PasswordHash, &u.Role, &u.PlanID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &u, nil
}

// GetUserByEmail finds a user by email address. Returns nil if not found.
func (r *UserRepo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, email, password_hash, role, plan_id, created_at, updated_at
		 FROM users
		 WHERE email = $1 AND is_deleted = 0`,
		email,
	).Scan(&u.UserID, &u.Email, &u.PasswordHash, &u.Role, &u.PlanID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
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
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, email, password_hash, role, plan_id, created_at, updated_at
		 FROM users
		 WHERE user_id = $1 AND is_deleted = 0`,
		userID,
	).Scan(&u.UserID, &u.Email, &u.PasswordHash, &u.Role, &u.PlanID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}
	return &u, nil
}
