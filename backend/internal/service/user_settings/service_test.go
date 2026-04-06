package user_settings

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
)

// fakeUserRepo is an in-memory stand-in for repo.UserRepo. It captures the
// last patch it received so tests can assert on forwarded arguments.
type fakeUserRepo struct {
	user     *model.User
	getErr   error
	patchErr error

	lastPatch *repo.UserSettingsPatch
}

func (f *fakeUserRepo) GetUserByID(_ context.Context, _ int64) (*model.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.user == nil {
		return nil, nil
	}
	copy := *f.user
	return &copy, nil
}

func (f *fakeUserRepo) UpdateUserSettings(
	_ context.Context, _ int64, patch *repo.UserSettingsPatch,
) (*model.User, error) {
	f.lastPatch = patch
	if f.patchErr != nil {
		return nil, f.patchErr
	}
	if f.user == nil {
		return nil, nil
	}
	// Apply the patch to the in-memory user so the returned snapshot reflects
	// the updated state.
	u := *f.user
	if patch != nil {
		if patch.ClearTotalCapitalCNY {
			u.TotalCapitalCNY = nil
		} else if patch.TotalCapitalCNY != nil {
			v := *patch.TotalCapitalCNY
			u.TotalCapitalCNY = &v
		}
		if patch.RiskPreference != nil {
			u.RiskPreference = *patch.RiskPreference
		}
		if patch.Categories != nil {
			cats := append([]string(nil), *patch.Categories...)
			u.Categories = cats
		}
	}
	f.user = &u
	return &u, nil
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

func TestGetUserSettings_Default(t *testing.T) {
	repo := &fakeUserRepo{user: baseUser()}
	s := NewService(repo)

	got, err := s.GetUserSettings(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != 42 {
		t.Errorf("UserID: want 42, got %d", got.UserID)
	}
	if got.RiskPreference != model.RiskPreferenceNeutral {
		t.Errorf("RiskPreference: want neutral, got %q", got.RiskPreference)
	}
	if got.TotalCapitalCNY != nil {
		t.Errorf("TotalCapitalCNY: want nil, got %v", *got.TotalCapitalCNY)
	}
	if got.OnboardingCompleted {
		t.Errorf("OnboardingCompleted: want false")
	}
	if len(got.Categories) != 0 {
		t.Errorf("Categories: want empty, got %v", got.Categories)
	}
}

func TestGetUserSettings_WithOnboardingStamp(t *testing.T) {
	u := baseUser()
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	u.OnboardingCompletedAt = &ts
	capital := 50000.0
	u.TotalCapitalCNY = &capital

	s := NewService(&fakeUserRepo{user: u})
	got, err := s.GetUserSettings(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.OnboardingCompleted {
		t.Errorf("OnboardingCompleted: want true")
	}
	if got.OnboardingCompletedAt == nil || *got.OnboardingCompletedAt != "2026-03-01T12:00:00Z" {
		t.Errorf("OnboardingCompletedAt: want 2026-03-01T12:00:00Z, got %v", got.OnboardingCompletedAt)
	}
	if got.TotalCapitalCNY == nil || *got.TotalCapitalCNY != 50000.0 {
		t.Errorf("TotalCapitalCNY: want 50000, got %v", got.TotalCapitalCNY)
	}
}

func TestGetUserSettings_NotFound(t *testing.T) {
	s := NewService(&fakeUserRepo{})
	_, err := s.GetUserSettings(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("want USER_NOT_FOUND, got %v", err)
	}
}

func TestGetUserSettings_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{getErr: errors.New("boom")})
	_, err := s.GetUserSettings(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPatchUserSettings_EachFieldIndividually(t *testing.T) {
	cap := 12345.67
	risk := model.RiskPreferenceAggressive
	cats := []string{model.AssetTypeGoldETF, model.AssetTypeAShareBroad}

	cases := []struct {
		name    string
		patch   *PatchUserSettings
		checkFn func(t *testing.T, got *UserSettings)
	}{
		{
			name:  "total capital only",
			patch: &PatchUserSettings{TotalCapitalCNY: &cap},
			checkFn: func(t *testing.T, got *UserSettings) {
				if got.TotalCapitalCNY == nil || *got.TotalCapitalCNY != cap {
					t.Errorf("TotalCapitalCNY: want %v, got %v", cap, got.TotalCapitalCNY)
				}
			},
		},
		{
			name:  "risk preference only",
			patch: &PatchUserSettings{RiskPreference: &risk},
			checkFn: func(t *testing.T, got *UserSettings) {
				if got.RiskPreference != risk {
					t.Errorf("RiskPreference: want %q, got %q", risk, got.RiskPreference)
				}
			},
		},
		{
			name:  "categories only",
			patch: &PatchUserSettings{Categories: &cats},
			checkFn: func(t *testing.T, got *UserSettings) {
				if len(got.Categories) != 2 {
					t.Errorf("Categories len: want 2, got %d", len(got.Categories))
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewService(&fakeUserRepo{user: baseUser()})
			got, err := s.PatchUserSettings(context.Background(), 42, tc.patch)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.checkFn(t, got)
		})
	}
}

func TestPatchUserSettings_CombinedPatch(t *testing.T) {
	cap := 10000.0
	risk := model.RiskPreferenceConservative
	cats := []string{model.AssetTypeUSStock}

	s := NewService(&fakeUserRepo{user: baseUser()})
	got, err := s.PatchUserSettings(context.Background(), 42, &PatchUserSettings{
		TotalCapitalCNY: &cap,
		RiskPreference:  &risk,
		Categories:      &cats,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RiskPreference != risk || *got.TotalCapitalCNY != cap || len(got.Categories) != 1 {
		t.Errorf("combined patch not applied: %+v", got)
	}
}

func TestPatchUserSettings_ClearTotalCapital(t *testing.T) {
	u := baseUser()
	cap := 99.0
	u.TotalCapitalCNY = &cap

	fr := &fakeUserRepo{user: u}
	s := NewService(fr)
	got, err := s.PatchUserSettings(context.Background(), 42,
		&PatchUserSettings{ClearTotalCapitalCNY: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCapitalCNY != nil {
		t.Errorf("expected nil after clear, got %v", *got.TotalCapitalCNY)
	}
	if !fr.lastPatch.ClearTotalCapitalCNY {
		t.Error("ClearTotalCapitalCNY flag not forwarded to repo")
	}
}

func TestPatchUserSettings_NilPatchIsNoop(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser()})
	got, err := s.PatchUserSettings(context.Background(), 42, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil settings")
	}
}

func TestPatchUserSettings_ValidationErrors(t *testing.T) {
	bad := "wild"
	neg := -1.0
	invalidCats := []string{"crypto"}
	dupCats := []string{model.AssetTypeGoldETF, model.AssetTypeGoldETF}

	cases := []struct {
		name string
		p    *PatchUserSettings
		code string
	}{
		{"negative capital", &PatchUserSettings{TotalCapitalCNY: &neg}, "INVALID_TOTAL_CAPITAL"},
		{"invalid risk", &PatchUserSettings{RiskPreference: &bad}, "INVALID_RISK_PREFERENCE"},
		{"invalid category", &PatchUserSettings{Categories: &invalidCats}, "INVALID_CATEGORY"},
		{"duplicate category", &PatchUserSettings{Categories: &dupCats}, "INVALID_CATEGORY"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewService(&fakeUserRepo{user: baseUser()})
			_, err := s.PatchUserSettings(context.Background(), 42, tc.p)
			if err == nil {
				t.Fatal("expected error")
			}
			var appErr *model.AppError
			if !errors.As(err, &appErr) || appErr.Code != tc.code {
				t.Errorf("want %q, got %v", tc.code, err)
			}
		})
	}
}

func TestPatchUserSettings_ZeroCapitalAllowed(t *testing.T) {
	zero := 0.0
	s := NewService(&fakeUserRepo{user: baseUser()})
	got, err := s.PatchUserSettings(context.Background(), 42,
		&PatchUserSettings{TotalCapitalCNY: &zero})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TotalCapitalCNY == nil || *got.TotalCapitalCNY != 0 {
		t.Errorf("expected 0 capital, got %v", got.TotalCapitalCNY)
	}
}

func TestPatchUserSettings_NotFound(t *testing.T) {
	s := NewService(&fakeUserRepo{})
	_, err := s.PatchUserSettings(context.Background(), 42, &PatchUserSettings{})
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) || appErr.Code != "USER_NOT_FOUND" {
		t.Errorf("want USER_NOT_FOUND, got %v", err)
	}
}

func TestPatchUserSettings_RepoError(t *testing.T) {
	s := NewService(&fakeUserRepo{user: baseUser(), patchErr: errors.New("db down")})
	// Use a non-empty patch so the service actually invokes UpdateUserSettings
	// and surfaces the repo error. Empty patches are short-circuited to GetUserSettings.
	pref := "aggressive"
	_, err := s.PatchUserSettings(context.Background(), 42, &PatchUserSettings{RiskPreference: &pref})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPatchUserSettings_EmptyPatchShortCircuit(t *testing.T) {
	// An empty patch must NOT trigger UpdateUserSettings. It should return the
	// current settings without bumping updated_at. The fake repo is poisoned so
	// that any UpdateUserSettings call would fail loudly.
	s := NewService(&fakeUserRepo{user: baseUser(), patchErr: errors.New("should not be called")})
	got, err := s.PatchUserSettings(context.Background(), 42, &PatchUserSettings{})
	if err != nil {
		t.Fatalf("unexpected error on empty patch: %v", err)
	}
	if got == nil || got.UserID != 42 {
		t.Fatalf("expected current settings, got %+v", got)
	}
}

func TestPatchUserSettings_ClearFlagOverridesValue(t *testing.T) {
	// When both ClearTotalCapitalCNY and TotalCapitalCNY are provided, the
	// clear flag must win and the provided value must be ignored.
	fake := &fakeUserRepo{user: baseUser()}
	s := NewService(fake)
	val := 100000.0
	_, err := s.PatchUserSettings(context.Background(), 42, &PatchUserSettings{
		TotalCapitalCNY:      &val,
		ClearTotalCapitalCNY: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.lastPatch == nil {
		t.Fatal("expected repo patch to be set")
	}
	if !fake.lastPatch.ClearTotalCapitalCNY {
		t.Error("expected ClearTotalCapitalCNY=true in repo patch")
	}
}
