package model

import "time"

// Plan represents a subscription plan that controls user quotas.
type Plan struct {
	PlanID           int64     `json:"planId"`
	Name             string    `json:"name"`
	MaxHoldings      int       `json:"maxHoldings"`
	MaxDailyAnalysis int       `json:"maxDailyAnalysis"`
	MaxPushChannels  int       `json:"maxPushChannels"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Creator          string    `json:"-"`
	Modifier         string    `json:"-"`
	IsDeleted        int16     `json:"-"`
}
