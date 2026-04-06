package wechat

import (
	"context"
	"encoding/json"

	"github.com/richman/backend/internal/notification/adapter"
	"go.uber.org/zap"
)

// wechatConfig holds WeChat-specific channel configuration.
type wechatConfig struct {
	OpenID     string `json:"openId"`
	TemplateID string `json:"templateId"`
}

// Adapter sends notifications via WeChat template messages.
// Current implementation is a stub that logs instead of making real API calls.
type Adapter struct {
	appID     string
	appSecret string
	logger    *zap.Logger
}

// New creates a new WeChat adapter.
func New(appID, appSecret string, logger *zap.Logger) *Adapter {
	return &Adapter{
		appID:     appID,
		appSecret: appSecret,
		logger:    logger,
	}
}

// Send delivers a message via WeChat template message.
// TODO: implement real WeChat API integration.
func (a *Adapter) Send(_ context.Context, config json.RawMessage, msg adapter.Message) error {
	var cfg wechatConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return err
	}

	a.logger.Info("wechat notification (stub)",
		zap.String("open_id", cfg.OpenID),
		zap.String("subject", msg.Subject),
		zap.String("summary", msg.CardSummary),
	)

	return nil
}

// Type returns the channel type identifier.
func (a *Adapter) Type() string {
	return "wechat"
}
