package model

import (
	"encoding/json"
	"time"
	"unicode/utf8"
)

// UserInviteCode maps to rm_user_invite_codes. Each user has up to 20
// personal invite codes (3 generated at registration, plus 1 per 7-day
// continuous login streak).
type UserInviteCode struct {
	InviteCodeID int64      `json:"inviteCodeId"`
	UserID       int64      `json:"userId"`
	Code         string     `json:"code"`
	IsUsed       bool       `json:"isUsed"`
	UsedByUserID *int64     `json:"usedByUserId,omitempty"`
	UsedAt       *time.Time `json:"usedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Creator      string     `json:"-"`
	Modifier     string     `json:"-"`
	IsDeleted    int16      `json:"-"`
}

// InviteReward maps to rm_invite_rewards. Records rewards granted to both
// inviter and invitee when a personal invite code is successfully used.
type InviteReward struct {
	RewardID       int64           `json:"rewardId"`
	UserID         int64           `json:"userId"`
	RewardType     string          `json:"rewardType"`
	RewardDetail   json.RawMessage `json:"rewardDetail,omitempty"`
	SourceInviteID int64           `json:"sourceInviteId"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Creator        string          `json:"-"`
	Modifier       string          `json:"-"`
	IsDeleted      int16           `json:"-"`
}

// InvitedUser represents a user invited via a personal invite code, with
// name masking applied for privacy. Returned by the GET /invite/my-invites
// endpoint.
type InvitedUser struct {
	InvitedUserID   int64     `json:"invitedUserId"`
	InvitedUserName string    `json:"invitedUserName"` // masked display name
	InvitedAt       time.Time `json:"invitedAt"`
}

// MaskName returns a privacy-safe display name showing only the first
// character of the name followed by asterisks. For example, "张三丰" becomes
// "张***". If the name is empty, returns "***".
func MaskName(name string) string {
	runes := []rune(name)
	if len(runes) == 0 {
		return "***"
	}
	// Keep only the first rune; replace the rest with asterisks.
	_ = utf8.RuneLen(runes[0]) // ensure valid UTF-8 handling
	masked := string(runes[0])
	for i := 1; i < len(runes); i++ {
		masked += "*"
	}
	return masked
}
