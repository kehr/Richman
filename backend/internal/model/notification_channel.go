package model

import (
	"encoding/json"
	"time"
)

// NotificationChannel represents a user's notification delivery channel.
type NotificationChannel struct {
	ChannelID   int64           `json:"channelId"`
	UserID      int64           `json:"userId"`
	ChannelType string          `json:"channelType"`
	Config      json.RawMessage `json:"config"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// CreateChannelInput contains the data required to create a notification channel.
type CreateChannelInput struct {
	ChannelType string          `json:"channelType" binding:"required"`
	Config      json.RawMessage `json:"config" binding:"required"`
}

// UpdateChannelInput contains the data allowed to be updated on a notification channel.
type UpdateChannelInput struct {
	Config  json.RawMessage `json:"config,omitempty"`
	Enabled *bool           `json:"enabled,omitempty"`
}
