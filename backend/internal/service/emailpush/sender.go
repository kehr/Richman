package emailpush

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	// batchSize is the maximum number of BCC recipients per SMTP call.
	batchSize = 50
	// batchDelay is the pause between consecutive batch sends to avoid throttling.
	batchDelay = 1 * time.Second
)

// Sender delivers email messages via SMTP.
type Sender struct {
	host     string
	port     int
	user     string
	password string
	from     string
	logger   *zap.Logger
}

// NewSender creates a Sender from explicit SMTP credentials.
// from should be a formatted address such as "Richman <noreply@richman.app>".
func NewSender(host string, port int, user, password, from string, logger *zap.Logger) *Sender {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Sender{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
		logger:   logger,
	}
}

// Send delivers a single email to one recipient.
// It logs a warning and returns nil when SMTP is not configured, so the
// caller treats an unconfigured mailer as a no-op rather than an error.
func (s *Sender) Send(ctx context.Context, to, subject, htmlBody string) error {
	if s.host == "" {
		s.logger.Warn("smtp not configured, skipping email",
			zap.String("to", to),
			zap.String("subject", subject),
		)
		return nil
	}

	msg := s.buildMessage([]string{to}, nil, subject, htmlBody)
	if err := s.dial(to, []string{to}, msg); err != nil {
		return fmt.Errorf("send email to %s: %w", to, err)
	}

	s.logger.Info("email sent", zap.String("to", to), zap.String("subject", subject))
	return nil
}

// SendBatch delivers the same email to multiple recipients using BCC grouping.
// Recipients are split into batches of batchSize with a 1-second pause between
// batches to avoid SMTP rate limits. Individual batch failures are logged and
// skipped so the overall send continues.
func (s *Sender) SendBatch(ctx context.Context, recipients []string, subject, htmlBody string) error {
	if s.host == "" {
		s.logger.Warn("smtp not configured, skipping batch email",
			zap.Int("count", len(recipients)),
			zap.String("subject", subject),
		)
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}

	total := len(recipients)
	sent := 0
	failed := 0

	for i := 0; i < total; i += batchSize {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during batch send: %w", ctx.Err())
		default:
		}

		end := i + batchSize
		if end > total {
			end = total
		}
		batch := recipients[i:end]

		// Use the first recipient as the visible To: address; the rest are BCC.
		msg := s.buildMessage([]string{batch[0]}, batch, subject, htmlBody)
		if err := s.dial(batch[0], batch, msg); err != nil {
			s.logger.Error("batch email failed",
				zap.Int("batch_start", i),
				zap.Int("batch_size", len(batch)),
				zap.String("subject", subject),
				zap.Error(err),
			)
			failed += len(batch)
		} else {
			sent += len(batch)
			s.logger.Info("batch email sent",
				zap.Int("batch_start", i),
				zap.Int("batch_size", len(batch)),
				zap.String("subject", subject),
			)
		}

		// Delay between batches except after the last one.
		if end < total {
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled between batches: %w", ctx.Err())
			case <-time.After(batchDelay):
			}
		}
	}

	s.logger.Info("batch email completed",
		zap.Int("total", total),
		zap.Int("sent", sent),
		zap.Int("failed", failed),
		zap.String("subject", subject),
	)
	return nil
}

// buildMessage constructs a raw MIME message with HTML body.
// bcc is an optional list of BCC addresses written into the Bcc header; all
// addresses in bcc will also receive the message via the SMTP RCPT TO envelope.
func (s *Sender) buildMessage(to []string, bcc []string, subject, htmlBody string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + s.from + "\r\n")
	sb.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	if len(bcc) > 0 {
		sb.WriteString("Bcc: " + strings.Join(bcc, ", ") + "\r\n")
	}
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(htmlBody)
	return []byte(sb.String())
}

// dial establishes an SMTP connection and delivers one message.
func (s *Sender) dial(from string, rcptTo []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	return smtp.SendMail(addr, auth, s.user, rcptTo, msg)
}
