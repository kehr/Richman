package model

import "time"

// InviteCode represents an invitation code used during registration.
type InviteCode struct {
	InviteCodeID int64     `json:"inviteCodeId"`
	Code         string    `json:"code"`
	MaxUses      int       `json:"maxUses"`
	UsedCount    int       `json:"usedCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Creator      string    `json:"-"`
	Modifier     string    `json:"-"`
	IsDeleted    int16     `json:"-"`
}
