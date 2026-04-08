// Package onboarding tracks whether a user has completed, skipped, or needs
// to re-run the product onboarding flow (PRD §2.3). The backend stores two
// mutually exclusive timestamps on the users table (onboarding_completed_at
// and onboarding_skipped_at); this service exposes a small read/write API
// consumed by the /api/v1/onboarding endpoints.
package onboarding

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/richman/backend/internal/model"
)

// UserRepo is the narrow data-access surface this service needs. It mirrors
// the subset of repo.UserRepo that deals with the onboarding_completed_at /
// onboarding_skipped_at columns so unit tests can stub the data layer without
// a live database.
type UserRepo interface {
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	MarkOnboardingCompleted(ctx context.Context, userID int64) (*model.User, error)
	MarkOnboardingSkipped(ctx context.Context, userID int64) (*model.User, error)
	ResetOnboarding(ctx context.Context, userID int64) (*model.User, error)
}

// Status is the read model returned to clients. The Completed/Skipped pair is
// mutually exclusive at the SQL layer: MarkOnboardingCompleted clears
// skipped_at and vice versa, so at most one of the two booleans can be true
// on any given response. Timestamps are serialized as RFC3339 strings to
// match the rest of the REST API.
type Status struct {
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Skipped     bool       `json:"skipped"`
	SkippedAt   *time.Time `json:"skippedAt,omitempty"`
}

// Service provides read/write access to the per-user onboarding state.
type Service struct {
	users UserRepo
}

// NewService constructs the onboarding Service. users must be non-nil; we
// fail fast at wiring time rather than defer the nil dereference to the first
// request.
func NewService(users UserRepo) *Service {
	if users == nil {
		panic("onboarding.NewService: users repository must not be nil")
	}
	return &Service{users: users}
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
// NULL and atomically clears any prior onboarding_skipped_at so the two
// flags stay mutually exclusive. Calling it on an already-completed user is
// a no-op and returns the existing timestamp.
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

// MarkSkipped stamps onboarding_skipped_at with NOW() if it is still NULL
// and atomically clears any prior onboarding_completed_at so the two flags
// stay mutually exclusive. Calling it on an already-skipped user is a no-op
// and returns the existing timestamp.
func (s *Service) MarkSkipped(ctx context.Context, userID int64) (*Status, error) {
	u, err := s.users.MarkOnboardingSkipped(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("mark onboarding skipped: %w", err)
	}
	if u == nil {
		return nil, model.NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}
	return statusFromUser(u), nil
}

// Reset clears both onboarding_completed_at and onboarding_skipped_at in a
// single atomic UPDATE, returning the user to the not-yet-onboarded state.
// This is a user-facing operation invoked from the Settings AccountTab CTA
// when the user wants to re-run the onboarding flow; no environment gating.
func (s *Service) Reset(ctx context.Context, userID int64) (*Status, error) {
	u, err := s.users.ResetOnboarding(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("reset onboarding: %w", err)
	}
	if u == nil {
		return nil, model.NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}
	return statusFromUser(u), nil
}

// statusFromUser projects a model.User into a Status DTO. Mutual exclusion
// between Completed and Skipped is enforced at the SQL layer, so this helper
// simply reflects whatever the user row contains without additional
// validation.
func statusFromUser(u *model.User) *Status {
	s := &Status{}
	if u.OnboardingCompletedAt != nil {
		ts := *u.OnboardingCompletedAt
		s.Completed = true
		s.CompletedAt = &ts
	}
	if u.OnboardingSkippedAt != nil {
		ts := *u.OnboardingSkippedAt
		s.Skipped = true
		s.SkippedAt = &ts
	}
	return s
}
