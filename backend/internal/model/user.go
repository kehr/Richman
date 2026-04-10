package model

import "time"

// Risk preference values. These must stay in sync with the CHECK constraint
// chk_users_risk_preference defined in migration 007_user_profile.up.sql.
const (
	RiskPreferenceConservative = "conservative"
	RiskPreferenceNeutral      = "neutral"
	RiskPreferenceAggressive   = "aggressive"
)

// Language preference values. These must stay in sync with the CHECK constraint
// chk_users_language defined in migration 014_user_language.
const (
	LanguageEN = "en"
	LanguageZH = "zh"
)

// Asset type values. These must stay in sync with the asset_catalog.asset_type
// column and the categories JSONB array on users (PRD §1.5).
const (
	AssetTypeGoldETF        = "gold_etf"
	AssetTypeAShareBroad    = "a_share_broad"
	AssetTypeAShareIndustry = "a_share_industry"
	AssetTypeUSStock        = "us_stock"
)

// User represents a registered user in the system.
type User struct {
	UserID                int64      `json:"userId"`
	Email                 string     `json:"email"`
	PasswordHash          string     `json:"-"`
	Role                  string     `json:"role"`
	PlanID                *int64     `json:"planId,omitempty"`
	RiskPreference        string     `json:"riskPreference"`
	TotalCapitalCNY       *float64   `json:"totalCapitalCny,omitempty"`
	OnboardingCompletedAt *time.Time `json:"onboardingCompletedAt,omitempty"`
	OnboardingSkippedAt   *time.Time `json:"onboardingSkippedAt,omitempty"`
	Categories            []string   `json:"categories"`
	Language              string     `json:"language"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
	Creator               string     `json:"-"`
	Modifier              string     `json:"-"`
	IsDeleted             int16      `json:"-"`
}
