// Package user_settings manages user-level profile configuration: total
// capital, risk preference, watched categories, and onboarding state.
//
// It also exposes two cross-cutting helpers used by the rest of the backend:
//
//   - money.AttachAmounts: amount projection done at the API response layer
//     only (TRD §5.3). The analysis / LLM / persistence layers never see
//     absolute amounts; they operate on percentages alone.
//   - privacy_guard.AssertNoCapitalLeakage: runtime assertion used to catch
//     accidental capital leakage into DTOs that must stay percentage-only
//     (TRD §5.2 runtime guard).
package user_settings

import (
	"context"
	"fmt"
	"net/http"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
)

// allowedRiskPreferences is the whitelist enforced at the service boundary.
// It mirrors the CHECK constraint defined in migration 007_user_profile.
var allowedRiskPreferences = map[string]struct{}{
	model.RiskPreferenceConservative: {},
	model.RiskPreferenceNeutral:      {},
	model.RiskPreferenceAggressive:   {},
}

// allowedLanguages mirrors the CHECK constraint chk_users_language in
// migration 014_user_language.
var allowedLanguages = map[string]struct{}{
	model.LanguageEN: {},
	model.LanguageZH: {},
}

// allowedDisplayCurrencies mirrors the CHECK constraint
// chk_users_display_currency in migration 016_user_display_currency.
var allowedDisplayCurrencies = map[string]struct{}{
	model.DisplayCurrencyCNY: {},
	model.DisplayCurrencyUSD: {},
	model.DisplayCurrencyHKD: {},
}

// allowedCategories mirrors the four asset types defined in PRD §1.5 and the
// asset_catalog.asset_type column.
var allowedCategories = map[string]struct{}{
	model.AssetTypeGoldETF:        {},
	model.AssetTypeAShareBroad:    {},
	model.AssetTypeAShareIndustry: {},
	model.AssetTypeUSStock:        {},
}

// UserRepo is the subset of repo.UserRepo that this service depends on. The
// interface exists so unit tests can stub the data layer without a live DB.
type UserRepo interface {
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	GetTotalCapitalCNY(ctx context.Context, userID int64) (*float64, error)
	UpdateUserSettings(
		ctx context.Context, userID int64, patch *repo.UserSettingsPatch,
	) (*model.User, error)
}

// UserSettings is the read-model exposed to callers. It mirrors the writable
// profile fields of model.User plus onboarding state.
type UserSettings struct {
	UserID                int64    `json:"userId"`
	TotalCapitalCNY       *float64 `json:"totalCapitalCny,omitempty"`
	RiskPreference        string   `json:"riskPreference"`
	Categories            []string `json:"categories"`
	Language              string   `json:"language"`
	DisplayCurrency       string   `json:"displayCurrency"`
	OnboardingCompleted   bool     `json:"onboardingCompleted"`
	OnboardingCompletedAt *string  `json:"onboardingCompletedAt,omitempty"`
}

// PatchUserSettings carries a sparse update. A nil pointer means "leave field
// unchanged". To clear the total capital back to NULL (e.g. a user opts out of
// private-mode capital tracking), set ClearTotalCapitalCNY = true; in that
// case TotalCapitalCNY is ignored.
type PatchUserSettings struct {
	TotalCapitalCNY      *float64  `json:"totalCapitalCny,omitempty"`
	ClearTotalCapitalCNY bool      `json:"clearTotalCapitalCny,omitempty"`
	RiskPreference       *string   `json:"riskPreference,omitempty"`
	Categories           *[]string `json:"categories,omitempty"`
	Language             *string   `json:"language,omitempty"`
	DisplayCurrency      *string   `json:"displayCurrency,omitempty"`
}

// Service provides read/write access to user profile settings.
type Service struct {
	users UserRepo
}

// NewService constructs the user_settings Service.
func NewService(users UserRepo) *Service {
	return &Service{users: users}
}

// GetTotalCapitalCNY loads only the user's optional total capital using a
// single-column SELECT to keep the hot path used by API handlers (decision_card,
// portfolio, etc.) light. Returns nil when the user does not exist or has
// not set a total capital.
func (s *Service) GetTotalCapitalCNY(ctx context.Context, userID int64) (*float64, error) {
	capital, err := s.users.GetTotalCapitalCNY(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load total capital: %w", err)
	}
	return capital, nil
}

// GetUserSettings loads the full settings snapshot for the given user.
func (s *Service) GetUserSettings(ctx context.Context, userID int64) (*UserSettings, error) {
	u, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}
	if u == nil {
		return nil, model.NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}
	return toUserSettings(u), nil
}

// PatchUserSettings validates the patch and applies it. Validation happens at
// the service boundary (not in the HTTP handler) so every caller gets the same
// guarantees.
func (s *Service) PatchUserSettings(
	ctx context.Context, userID int64, patch *PatchUserSettings,
) (*UserSettings, error) {
	if patch == nil {
		patch = &PatchUserSettings{}
	}

	if err := validatePatch(patch); err != nil {
		return nil, err
	}

	// Short-circuit no-op patches: if the caller provided an empty patch object
	// (no fields set, no clear flag), skip the UPDATE round trip and avoid
	// bumping updated_at on a true no-op call. Return the current settings.
	if isEmptyPatch(patch) {
		return s.GetUserSettings(ctx, userID)
	}

	repoPatch := &repo.UserSettingsPatch{
		TotalCapitalCNY:      patch.TotalCapitalCNY,
		ClearTotalCapitalCNY: patch.ClearTotalCapitalCNY,
		RiskPreference:       patch.RiskPreference,
		Categories:           patch.Categories,
		Language:             patch.Language,
		DisplayCurrency:      patch.DisplayCurrency,
	}

	u, err := s.users.UpdateUserSettings(ctx, userID, repoPatch)
	if err != nil {
		return nil, fmt.Errorf("update user settings: %w", err)
	}
	if u == nil {
		return nil, model.NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "user not found")
	}
	return toUserSettings(u), nil
}

// validatePatch enforces domain rules before touching the database.
func validatePatch(patch *PatchUserSettings) error {
	// Reject the contradictory combination "clear and set at the same time"
	// explicitly rather than silently preferring one over the other. This
	// matches the intent documented on PatchUserSettings.
	if patch.ClearTotalCapitalCNY && patch.TotalCapitalCNY != nil {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_TOTAL_CAPITAL",
			"total_capital_cny cannot be set and cleared in the same patch")
	}
	if patch.TotalCapitalCNY != nil && !patch.ClearTotalCapitalCNY {
		if *patch.TotalCapitalCNY < 0 {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_TOTAL_CAPITAL",
				"total_capital_cny must be >= 0")
		}
	}

	if patch.RiskPreference != nil {
		if _, ok := allowedRiskPreferences[*patch.RiskPreference]; !ok {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_RISK_PREFERENCE",
				fmt.Sprintf("risk_preference %q is not allowed", *patch.RiskPreference))
		}
	}

	if patch.Language != nil {
		if _, ok := allowedLanguages[*patch.Language]; !ok {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_LANGUAGE",
				fmt.Sprintf("language %q is not allowed", *patch.Language))
		}
	}

	if patch.DisplayCurrency != nil {
		if _, ok := allowedDisplayCurrencies[*patch.DisplayCurrency]; !ok {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_DISPLAY_CURRENCY",
				fmt.Sprintf("display_currency %q is not allowed", *patch.DisplayCurrency))
		}
	}

	if patch.Categories != nil {
		seen := make(map[string]struct{}, len(*patch.Categories))
		for _, c := range *patch.Categories {
			if _, ok := allowedCategories[c]; !ok {
				return model.NewAppError(http.StatusBadRequest,
					"INVALID_CATEGORY",
					fmt.Sprintf("category %q is not allowed", c))
			}
			if _, dup := seen[c]; dup {
				return model.NewAppError(http.StatusBadRequest,
					"INVALID_CATEGORY",
					fmt.Sprintf("category %q duplicated", c))
			}
			seen[c] = struct{}{}
		}
	}

	return nil
}

// isEmptyPatch reports whether the patch carries no fields to apply. An
// empty patch should not produce an UPDATE round trip or bump updated_at.
func isEmptyPatch(patch *PatchUserSettings) bool {
	if patch == nil {
		return true
	}
	return patch.TotalCapitalCNY == nil &&
		!patch.ClearTotalCapitalCNY &&
		patch.RiskPreference == nil &&
		patch.Categories == nil &&
		patch.Language == nil &&
		patch.DisplayCurrency == nil
}

// toUserSettings projects a model.User into the service-level DTO.
func toUserSettings(u *model.User) *UserSettings {
	out := &UserSettings{
		UserID:              u.UserID,
		TotalCapitalCNY:     u.TotalCapitalCNY,
		RiskPreference:      u.RiskPreference,
		Categories:          u.Categories,
		Language:            u.Language,
		DisplayCurrency:     u.DisplayCurrency,
		OnboardingCompleted: u.OnboardingCompletedAt != nil,
	}
	if out.Categories == nil {
		out.Categories = []string{}
	}
	if u.OnboardingCompletedAt != nil {
		s := u.OnboardingCompletedAt.UTC().Format("2006-01-02T15:04:05Z")
		out.OnboardingCompletedAt = &s
	}
	return out
}
