package adapter

import (
	"context"
	"encoding/json"
)

// Message holds the content to be delivered through a notification channel.
type Message struct {
	Subject     string // email subject or card title
	Body        string // HTML or plain text body
	CardSummary string // short summary for card-style messages
	UserEmail   string // recipient email for email channel
}

// Adapter defines the interface for sending notifications through a specific channel.
type Adapter interface {
	// Send delivers a message using the channel-specific configuration.
	Send(ctx context.Context, config json.RawMessage, msg Message) error
	// Type returns the channel type identifier (e.g. "wechat", "feishu", "email").
	Type() string
}
