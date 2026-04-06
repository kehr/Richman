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

// Service handles notification business logic.
type Service struct {
	channelRepo *repo.NotificationChannelRepo
	logRepo     *repo.NotificationLogRepo
	dispatcher  *notification.Dispatcher
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
