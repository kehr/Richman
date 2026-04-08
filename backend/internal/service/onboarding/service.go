// Package onboarding tracks whether a user has completed the product
// onboarding flow (PRD §2.3). The backend stores a single timestamp on the
// users table; this service exposes a small read/write API consumed by the
// /api/v1/onboarding endpoints and guarded by the dev-only reset switch
// described in PRD §6.2.
package onboarding

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/richman/backend/internal/model"
)

// UserRepo is the narrow data-access surface this service needs. It mirrors
// the subset of repo.UserRepo that deals with the onboarding_completed_at
// column so unit tests can stub the data layer without a live database.
type UserRepo interface {
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	MarkOnboardingCompleted(ctx context.Context, userID int64) (*model.User, error)
	ClearOnboardingCompleted(ctx context.Context, userID int64) (*model.User, error)
}

// EnvGuard reports whether the current runtime is production. Reset is only
// allowed when this returns false so an accidental call in production is
// rejected with 403 by the handler layer.
type EnvGuard interface {
	IsProduction() bool
}

// Status is the read model returned to clients. CompletedAt is only set when
// Completed is true and is serialized as an RFC3339 string to match the rest
// of the REST API.
type Status struct {
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// Service provides read/write access to the per-user onboarding state.
type Service struct {
	users UserRepo
	env   EnvGuard
}

// NewService constructs the onboarding Service. Both users and env must be
// non-nil; pass *config.Config directly for env since it already implements
// EnvGuard via IsProduction(). A nil env would make Reset always-allowed,
// which would defeat the production guard, so we fail fast at wiring time
// rather than silently open the hole.
func NewService(users UserRepo, env EnvGuard) *Service {
	if users == nil {
		panic("onboarding.NewService: users repository must not be nil")
	}
	if env == nil {
		panic("onboarding.NewService: env guard must not be nil")
	}
	return &Service{users: users, env: env}
}

// GetStatus loads the current onboarding state for the given user. Returns
// a USER_NOT_FOUND AppError (404) if the user does not exist.
func (s *Service) GetStatus(ctx context.Context, userID int64) (*Status, error) {
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}
	if u == nil {
		return nil, model.NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}
	return statusFromUser(u), nil
}

// MarkCompleted stamps onboarding_completed_at with NOW() if it is still
// NULL. Calling it on an already-completed user is a no-op and returns the
// existing timestamp (see repo.UserRepo.MarkOnboardingCompleted).
func (s *Service) MarkCompleted(ctx context.Context, userID int64) (*Status, error) {
	u, err := s.users.MarkOnboardingCompleted(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("mark onboarding completed: %w", err)
	}
	if u == nil {
		return nil, model.NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}
	return statusFromUser(u), nil
}

// Reset clears onboarding_completed_at back to NULL. It is only permitted in
// non-production environments; callers from production get a
// ONBOARDING_RESET_FORBIDDEN AppError mapped to 403 by the handler. The env
// check runs first so a missing user in production still returns 403 rather
// than leaking the existence of the user via 404.
func (s *Service) Reset(ctx context.Context, userID int64) (*Status, error) {
	if s.env.IsProduction() {
		return nil, model.NewAppError(
			http.StatusForbidden,
			"ONBOARDING_RESET_FORBIDDEN",
			"onboarding reset is not allowed in production",
		)
	}
	u, err := s.users.ClearOnboardingCompleted(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("clear onboarding completed: %w", err)
	}
	if u == nil {
		return nil, model.NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}
	return statusFromUser(u), nil
}

// statusFromUser projects a model.User into a Status DTO.
func statusFromUser(u *model.User) *Status {
	if u.OnboardingCompletedAt == nil {
		return &Status{Completed: false}
	}
	ts := *u.OnboardingCompletedAt
	return &Status{Completed: true, CompletedAt: &ts}
}
