package onboarding

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/richman/backend/internal/model"
)

// fakeUserRepo is an in-memory stand-in that applies mark/clear mutations to
// an embedded *model.User so GetStatus observes the updated state.
type fakeUserRepo struct {
	user     *model.User
	getErr   error
	markErr  error
	clearErr error
}

func (f *fakeUserRepo) GetUserByID(_ context.Context, _ int64) (*model.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.user == nil {
		return nil, nil
	}
	cp := *f.user
	return &cp, nil
}

func (f *fakeUserRepo) MarkOnboardingCompleted(_ context.Context, _ int64) (*model.User, error) {
	if f.markErr != nil {
		return nil, f.markErr
	}
	if f.user == nil {
		return nil, nil
	}
	if f.user.OnboardingCompletedAt == nil {
		now := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
		f.user.OnboardingCompletedAt = &now
	}
	cp := *f.user
	return &cp, nil
}

func (f *fakeUserRepo) ClearOnboardingCompleted(_ context.Context, _ int64) (*model.User, error) {
	if f.clearErr != nil {
		return nil, f.clearErr
	}
	if f.user == nil {
		return nil, nil
	}
	f.user.OnboardingCompletedAt = nil
	cp := *f.user
	return &cp, nil
}

type fakeEnv struct{ prod bool }

func (f fakeEnv) IsProduction() bool { return f.prod }

func baseUser() *model.User {
	return &model.User{
		UserID:         42,
		Email:          "alice@example.com",
		Role:           "user",
		RiskPreference: model.RiskPreferenceNeutral,
		Categories:     []string{},
	}
}

func TestGetStatus_DefaultIncomplete(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser()}, fakeEnv{prod: false})
	got, err := s.GetStatus(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Completed {
		t.Errorf("Completed: want false, got true")
	}
	if got.CompletedAt != nil {
		t.Errorf("CompletedAt: want nil, got %v", got.CompletedAt)
	}
}

func TestGetStatus_AlreadyCompleted(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	s := NewService(&fakeUserRepo{user: u}, fakeEnv{prod: true})

	got, err := s.GetStatus(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Completed {
		t.Errorf("Completed: want true")
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(ts) {
		t.Errorf("CompletedAt: want %v, got %v", ts, got.CompletedAt)
	}
}

func TestGetStatus_NotFound(t *testing.T) {
	s := NewService(&fakeUserRepo{}, fakeEnv{prod: false})
	_, err := s.GetStatus(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" || appErr.StatusCode != 404 {
		t.Errorf("want USER_NOT_FOUND 404, got %v", err)
	}
}

func TestGetStatus_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{getErr: errors.New("boom")}, fakeEnv{prod: false})
	_, err := s.GetStatus(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMarkCompleted_StampsTimestamp(t *testing.T) {
	repo := &fakeUserRepo{user: baseUser()}
	s := NewService(repo, fakeEnv{prod: true})

	got, err := s.MarkCompleted(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Completed || got.CompletedAt == nil {
		t.Fatalf("expected completed status with timestamp, got %+v", got)
	}
	// Subsequent GetStatus must observe the persisted timestamp.
	st, err := s.GetStatus(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !st.Completed {
		t.Errorf("GetStatus Completed: want true after MarkCompleted")
	}
}

func TestMarkCompleted_NotFound(t *testing.T) {
	s := NewService(&fakeUserRepo{}, fakeEnv{prod: false})
	_, err := s.MarkCompleted(context.Background(), 1)
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("want USER_NOT_FOUND, got %v", err)
	}
}

func TestMarkCompleted_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser(), markErr: errors.New("db")}, fakeEnv{prod: false})
	if _, err := s.MarkCompleted(context.Background(), 42); err == nil {
		t.Fatal("expected error")
	}
}

func TestReset_DevAllowed(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	repo := &fakeUserRepo{user: u}
	s := NewService(repo, fakeEnv{prod: false})

	got, err := s.Reset(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Completed {
		t.Errorf("Completed: want false after reset, got true")
	}
	if got.CompletedAt != nil {
		t.Errorf("CompletedAt: want nil after reset, got %v", got.CompletedAt)
	}
}

func TestReset_ProductionForbidden(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	s := NewService(&fakeUserRepo{user: u}, fakeEnv{prod: true})

	_, err := s.Reset(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error in production")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != 403 || appErr.Code != "ONBOARDING_RESET_FORBIDDEN" {
		t.Errorf("want 403 ONBOARDING_RESET_FORBIDDEN, got %v", err)
	}
}

func TestReset_NotFound(t *testing.T) {
	s := NewService(&fakeUserRepo{}, fakeEnv{prod: false})
	_, err := s.Reset(context.Background(), 1)
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("want USER_NOT_FOUND, got %v", err)
	}
}

func TestReset_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser(), clearErr: errors.New("db")}, fakeEnv{prod: false})
	if _, err := s.Reset(context.Background(), 42); err == nil {
		t.Fatal("expected error")
	}
}
