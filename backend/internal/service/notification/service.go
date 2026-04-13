package notification

import (
	"context"
	"fmt"
	"net/http"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/notification"
	"github.com/richman/backend/internal/notification/adapter"
	"github.com/richman/backend/internal/repo"
	"go.uber.org/zap"
)

// BulkEmailSender is a minimal interface for sending bulk HTML email to a list
// of recipients. The emailpush.Sender type satisfies this interface.
type BulkEmailSender interface {
	SendBatch(ctx context.Context, recipients []string, subject, htmlBody string) error
}

// Service handles notification business logic.
type Service struct {
	channelRepo *repo.NotificationChannelRepo
	logRepo     *repo.NotificationLogRepo
	dispatcher  *notification.Dispatcher
	bulkSender  BulkEmailSender // optional; used by SendBroadcast
	logger      *zap.Logger
}

// NewService creates a new notification Service.
func NewService(
	channelRepo *repo.NotificationChannelRepo,
	logRepo *repo.NotificationLogRepo,
	dispatcher *notification.Dispatcher,
	logger *zap.Logger,
) *Service {
	return &Service{
		channelRepo: channelRepo,
		logRepo:     logRepo,
		dispatcher:  dispatcher,
		logger:      logger,
	}
}

// WithBulkSender sets the bulk email sender used by SendBroadcast. This is
// optional; if not set, SendBroadcast logs a warning and returns nil.
func (s *Service) WithBulkSender(sender BulkEmailSender) *Service {
	s.bulkSender = sender
	return s
}

// ListChannels returns all notification channels for a user.
func (s *Service) ListChannels(ctx context.Context, userID int64) ([]model.NotificationChannel, error) {
	channels, err := s.channelRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	return channels, nil
}

// CreateChannel creates a new notification channel.
func (s *Service) CreateChannel(
	ctx context.Context, userID int64,
	input *model.CreateChannelInput, email string,
) (*model.NotificationChannel, error) {
	ch, err := s.channelRepo.Create(ctx, userID, input, email)
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}
	return ch, nil
}

// UpdateChannel updates an existing notification channel owned by the user.
func (s *Service) UpdateChannel(
	ctx context.Context, userID, channelID int64,
	input *model.UpdateChannelInput, email string,
) (*model.NotificationChannel, error) {
	existing, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("get channel: %w", err)
	}
	if existing == nil || existing.UserID != userID {
		return nil, model.ErrNotFound
	}

	ch, err := s.channelRepo.Update(ctx, channelID, input, email)
	if err != nil {
		return nil, fmt.Errorf("update channel: %w", err)
	}
	return ch, nil
}

// DeleteChannel soft-deletes a notification channel owned by the user.
func (s *Service) DeleteChannel(ctx context.Context, userID, channelID int64, email string) error {
	existing, err := s.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}
	if existing == nil || existing.UserID != userID {
		return model.ErrNotFound
	}

	if err := s.channelRepo.SoftDelete(ctx, channelID, email); err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}
	return nil
}

// SendToUser fetches the user's enabled channels and dispatches the message.
// messageType is used for logging (e.g. "am_brief", "pm_digest", "us_digest").
func (s *Service) SendToUser(
	ctx context.Context, userID int64,
	msg adapter.Message, messageType string,
) error {
	channels, err := s.channelRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list channels for send: %w", err)
	}

	if len(channels) == 0 {
		s.logger.Info("no notification channels configured",
			zap.Int64("user_id", userID),
		)
		return model.NewAppError(http.StatusBadRequest, "NO_CHANNELS",
			"no notification channels configured")
	}

	results := s.dispatcher.Send(ctx, channels, msg)

	// Log each result.
	for i := range results {
		r := &results[i]
		status := "sent"
		errMsg := ""
		if !r.Success {
			status = "failed"
			if r.Error != nil {
				errMsg = r.Error.Error()
			}
		}

		_, logErr := s.logRepo.Create(ctx, userID, r.ChannelType, messageType, status, errMsg)
		if logErr != nil {
			s.logger.Warn("failed to log notification result",
				zap.Int64("user_id", userID),
				zap.String("channel_type", r.ChannelType),
				zap.Error(logErr),
			)
		}
	}

	return nil
}

// SendToUsers sends a notification to multiple users. Used by cron jobs.
func (s *Service) SendToUsers(
	ctx context.Context, userIDs []int64,
	msgBuilder func(userID int64) *adapter.Message,
	messageType string,
) {
	for _, uid := range userIDs {
		msg := msgBuilder(uid)
		if msg == nil {
			continue
		}
		if err := s.SendToUser(ctx, uid, *msg, messageType); err != nil {
			s.logger.Warn("failed to send notification to user",
				zap.Int64("user_id", uid),
				zap.String("message_type", messageType),
				zap.Error(err),
			)
		}
	}
}

// SendBroadcast delivers an HTML email to a list of recipients using the
// configured BulkEmailSender. It is a thin wrapper that logs progress and
// delegates batching to the sender implementation.
//
// Returns nil without sending when no bulk sender has been configured
// (WithBulkSender was not called), so the caller does not need to guard.
func (s *Service) SendBroadcast(
	ctx context.Context, recipients []string, subject, htmlBody string,
) error {
	if s.bulkSender == nil {
		s.logger.Warn("SendBroadcast called but no bulk sender configured; skipping",
			zap.String("subject", subject),
			zap.Int("recipients", len(recipients)),
		)
		return nil
	}

	s.logger.Info("broadcasting email",
		zap.String("subject", subject),
		zap.Int("recipients", len(recipients)),
	)

	if err := s.bulkSender.SendBatch(ctx, recipients, subject, htmlBody); err != nil {
		return fmt.Errorf("broadcast email: %w", err)
	}
	return nil
}
