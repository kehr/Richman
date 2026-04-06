package model

import "time"

// Risk preference values. These must stay in sync with the CHECK constraint
// chk_users_risk_preference defined in migration 007_user_profile.up.sql.
const (
	RiskPreferenceConservative = "conservative"
	RiskPreferenceNeutral      = "neutral"
	RiskPreferenceAggressive   = "aggressive"
)

// User represents a registered user in the system.
type User struct {
	UserID         int64     `json:"userId"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"-"`
	Role           string    `json:"role"`
	PlanID         *int64    `json:"planId,omitempty"`
	RiskPreference string    `json:"riskPreference"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Creator        string    `json:"-"`
	Modifier       string    `json:"-"`
	IsDeleted      int16     `json:"-"`
}
