package model

import "time"

// NotificationLog records the result of a notification delivery attempt.
type NotificationLog struct {
	LogID        int64     `json:"logId"`
	UserID       int64     `json:"userId"`
	ChannelType  string    `json:"channelType"`
	MessageType  string    `json:"messageType"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
