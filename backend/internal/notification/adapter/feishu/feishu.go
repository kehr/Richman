package feishu

import (
	"context"
	"encoding/json"

	"github.com/richman/backend/internal/notification/adapter"
	"go.uber.org/zap"
)

// feishuConfig holds Feishu-specific channel configuration.
type feishuConfig struct {
	WebhookURL string `json:"webhookUrl"`
}

// Adapter sends notifications via Feishu webhook card messages.
// Current implementation is a stub that logs instead of making real API calls.
type Adapter struct {
	defaultWebhook string
	logger         *zap.Logger
}

// New creates a new Feishu adapter.
func New(defaultWebhook string, logger *zap.Logger) *Adapter {
	return &Adapter{
		defaultWebhook: defaultWebhook,
		logger:         logger,
	}
}

// Send delivers a message via Feishu webhook.
// TODO: implement real Feishu webhook API integration.
func (a *Adapter) Send(_ context.Context, config json.RawMessage, msg adapter.Message) error {
	var cfg feishuConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return err
	}

	webhook := cfg.WebhookURL
	if webhook == "" {
		webhook = a.defaultWebhook
	}

	a.logger.Info("feishu notification (stub)",
		zap.String("webhook", webhook),
		zap.String("subject", msg.Subject),
		zap.String("summary", msg.CardSummary),
	)

	return nil
}

// Type returns the channel type identifier.
func (a *Adapter) Type() string {
	return "feishu"
}
