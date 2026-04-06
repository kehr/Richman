package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/richman/backend/internal/notification/adapter"
	"go.uber.org/zap"
)

// emailConfig holds email-specific channel configuration.
type emailConfig struct {
	To string `json:"to"` // recipient override; falls back to msg.UserEmail
}

// Adapter sends notifications via SMTP email.
type Adapter struct {
	host     string
	port     int
	user     string
	password string
	logger   *zap.Logger
}

// New creates a new email adapter.
func New(host string, port int, user, password string, logger *zap.Logger) *Adapter {
	return &Adapter{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		logger:   logger,
	}
}

// Send delivers a message via SMTP email.
func (a *Adapter) Send(_ context.Context, config json.RawMessage, msg adapter.Message) error {
	var cfg emailConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse email config: %w", err)
	}

	to := cfg.To
	if to == "" {
		to = msg.UserEmail
	}
	if to == "" {
		return fmt.Errorf("no recipient email address")
	}

	// If SMTP is not configured, log and skip.
	if a.host == "" {
		a.logger.Warn("smtp not configured, skipping email",
			zap.String("to", to),
			zap.String("subject", msg.Subject),
		)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", a.host, a.port)
	auth := smtp.PlainAuth("", a.user, a.password, a.host)

	headers := strings.Join([]string{
		fmt.Sprintf("From: %s", a.user),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", msg.Subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}, "\r\n")

	body := headers + "\r\n\r\n" + msg.Body

	if err := smtp.SendMail(addr, auth, a.user, []string{to}, []byte(body)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	a.logger.Info("email sent",
		zap.String("to", to),
		zap.String("subject", msg.Subject),
	)

	return nil
}

// Type returns the channel type identifier.
func (a *Adapter) Type() string {
	return "email"
}
