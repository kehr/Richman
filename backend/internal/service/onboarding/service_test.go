package onboarding

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/richman/backend/internal/model"
)

// fakeUserRepo is an in-memory stand-in that applies mark/clear mutations to
// an embedded *model.User so GetStatus observes the updated state. It mirrors
// the mutual-exclusion semantics of the real SQL layer: MarkOnboardingCompleted
// clears skipped_at and MarkOnboardingSkipped clears completed_at.
type fakeUserRepo struct {
	user       *model.User
	getErr     error
	markErr    error
	skipErr    error
	clearErr   error
	completeAt time.Time
	skipAt     time.Time
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
	// COALESCE semantics: only stamp if currently NULL.
	if f.user.OnboardingCompletedAt == nil {
		ts := f.completeAt
		if ts.IsZero() {
			ts = time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
		}
		f.user.OnboardingCompletedAt = &ts
	}
	// Atomic clear of the opposite field.
	f.user.OnboardingSkippedAt = nil
	cp := *f.user
	return &cp, nil
}

func (f *fakeUserRepo) MarkOnboardingSkipped(_ context.Context, _ int64) (*model.User, error) {
	if f.skipErr != nil {
		return nil, f.skipErr
	}
	if f.user == nil {
		return nil, nil
	}
	// COALESCE semantics: only stamp if currently NULL.
	if f.user.OnboardingSkippedAt == nil {
		ts := f.skipAt
		if ts.IsZero() {
			ts = time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC)
		}
		f.user.OnboardingSkippedAt = &ts
	}
	// Atomic clear of the opposite field.
	f.user.OnboardingCompletedAt = nil
	cp := *f.user
	return &cp, nil
}

func (f *fakeUserRepo) ResetOnboarding(_ context.Context, _ int64) (*model.User, error) {
	if f.clearErr != nil {
		return nil, f.clearErr
	}
	if f.user == nil {
		return nil, nil
	}
	f.user.OnboardingCompletedAt = nil
	f.user.OnboardingSkippedAt = nil
	cp := *f.user
	return &cp, nil
}

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
	s := NewService(&fakeUserRepo{user: baseUser()})
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
	if got.Skipped {
		t.Errorf("Skipped: want false, got true")
	}
	if got.SkippedAt != nil {
		t.Errorf("SkippedAt: want nil, got %v", got.SkippedAt)
	}
}

func TestGetStatus_AlreadyCompleted(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	s := NewService(&fakeUserRepo{user: u})

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
	s := NewService(&fakeUserRepo{})
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
	s := NewService(&fakeUserRepo{getErr: errors.New("boom")})
	_, err := s.GetStatus(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetStatus_ReflectsBothFields verifies statusFromUser projects
// skipped_at alongside completed_at so API consumers see the full state.
func TestGetStatus_ReflectsBothFields(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	u.OnboardingSkippedAt = &ts
	s := NewService(&fakeUserRepo{user: u})

	got, err := s.GetStatus(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Completed {
		t.Errorf("Completed: want false, got true")
	}
	if !got.Skipped {
		t.Errorf("Skipped: want true")
	}
	if got.SkippedAt == nil || !got.SkippedAt.Equal(ts) {
		t.Errorf("SkippedAt: want %v, got %v", ts, got.SkippedAt)
	}
}

func TestMarkCompleted_StampsTimestamp(t *testing.T) {
	repo := &fakeUserRepo{user: baseUser()}
	s := NewService(repo)

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
	s := NewService(&fakeUserRepo{})
	_, err := s.MarkCompleted(context.Background(), 1)
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("want USER_NOT_FOUND, got %v", err)
	}
}

func TestMarkCompleted_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser(), markErr: errors.New("db")})
	if _, err := s.MarkCompleted(context.Background(), 42); err == nil {
		t.Fatal("expected error")
	}
}

// TestMarkCompleted_ClearsSkipped verifies the mutual-exclusion guarantee:
// when a skipped user calls MarkCompleted, the skipped_at field is cleared.
func TestMarkCompleted_ClearsSkipped(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	u.OnboardingSkippedAt = &ts
	s := NewService(&fakeUserRepo{user: u})

	got, err := s.MarkCompleted(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Completed || got.CompletedAt == nil {
		t.Errorf("expected Completed=true with timestamp, got %+v", got)
	}
	if got.Skipped {
		t.Errorf("expected Skipped=false after MarkCompleted, got %+v", got)
	}
	if got.SkippedAt != nil {
		t.Errorf("expected SkippedAt=nil after MarkCompleted, got %v", got.SkippedAt)
	}
}

// TestMarkSkipped_Idempotent verifies COALESCE semantics: calling MarkSkipped
// twice returns the same timestamp because the second call does not overwrite
// the existing skipped_at value.
func TestMarkSkipped_Idempotent(t *testing.T) {
	repo := &fakeUserRepo{user: baseUser()}
	s := NewService(repo)

	first, err := s.MarkSkipped(context.Background(), 42)
	if err != nil {
		t.Fatalf("first MarkSkipped: %v", err)
	}
	if !first.Skipped || first.SkippedAt == nil {
		t.Fatalf("expected Skipped=true with timestamp, got %+v", first)
	}
	firstTS := *first.SkippedAt

	second, err := s.MarkSkipped(context.Background(), 42)
	if err != nil {
		t.Fatalf("second MarkSkipped: %v", err)
	}
	if !second.Skipped || second.SkippedAt == nil {
		t.Fatalf("expected Skipped=true with timestamp, got %+v", second)
	}
	if !second.SkippedAt.Equal(firstTS) {
		t.Errorf("SkippedAt: want unchanged %v, got %v", firstTS, second.SkippedAt)
	}
}

// TestMarkSkipped_ClearsCompleted verifies the mutual-exclusion guarantee:
// when a completed user calls MarkSkipped, the completed_at field is cleared.
func TestMarkSkipped_ClearsCompleted(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	s := NewService(&fakeUserRepo{user: u})

	got, err := s.MarkSkipped(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Completed {
		t.Errorf("expected Completed=false after MarkSkipped, got %+v", got)
	}
	if got.CompletedAt != nil {
		t.Errorf("expected CompletedAt=nil after MarkSkipped, got %v", got.CompletedAt)
	}
	if !got.Skipped || got.SkippedAt == nil {
		t.Errorf("expected Skipped=true with timestamp, got %+v", got)
	}
}

func TestMarkSkipped_NotFound(t *testing.T) {
	s := NewService(&fakeUserRepo{})
	_, err := s.MarkSkipped(context.Background(), 1)
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("want USER_NOT_FOUND, got %v", err)
	}
}

func TestMarkSkipped_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser(), skipErr: errors.New("db")})
	if _, err := s.MarkSkipped(context.Background(), 42); err == nil {
		t.Fatal("expected error")
	}
}

// TestReset_ClearsBothColumns verifies Reset atomically wipes both
// completed_at and skipped_at back to NULL in a single call.
func TestReset_ClearsBothColumns(t *testing.T) {
	u := baseUser()
	completedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	skippedAt := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &completedAt
	u.OnboardingSkippedAt = &skippedAt
	s := NewService(&fakeUserRepo{user: u})

	got, err := s.Reset(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Completed || got.CompletedAt != nil {
		t.Errorf("expected Completed cleared, got %+v", got)
	}
	if got.Skipped || got.SkippedAt != nil {
		t.Errorf("expected Skipped cleared, got %+v", got)
	}
}

// TestReset_AllowedInProduction proves the former production guard is gone:
// Reset succeeds regardless of environment because it is now a user-facing
// operation (Settings AccountTab CTA) rather than a dev-only debug shortcut.
// NewService no longer takes an env argument at all; the absence of that
// parameter is itself the assertion that the guard was removed.
func TestReset_AllowedInProduction(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	s := NewService(&fakeUserRepo{user: u})

	got, err := s.Reset(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Completed {
		t.Errorf("Completed: want false after reset, got true")
	}
}

func TestReset_NotFound(t *testing.T) {
	s := NewService(&fakeUserRepo{})
	_, err := s.Reset(context.Background(), 1)
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("want USER_NOT_FOUND, got %v", err)
	}
}

func TestReset_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser(), clearErr: errors.New("db")})
	if _, err := s.Reset(context.Background(), 42); err == nil {
		t.Fatal("expected error")
	}
}

// TestNewService_PanicsOnNilDeps verifies the fail-fast wiring guard: a nil
// users parameter must panic at construction time so a production process
// cannot start with a broken onboarding service.
func TestNewService_PanicsOnNilDeps(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	_ = NewService(nil)
}
