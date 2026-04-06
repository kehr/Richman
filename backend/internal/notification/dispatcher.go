package notification

import (
	"context"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/notification/adapter"
	"go.uber.org/zap"
)

// SendResult holds the outcome of a single channel delivery attempt.
type SendResult struct {
	ChannelType string
	Success     bool
	Error       error
}

// Dispatcher routes messages to the appropriate notification adapters.
type Dispatcher struct {
	adapters map[string]adapter.Adapter
	logger   *zap.Logger
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(logger *zap.Logger) *Dispatcher {
	return &Dispatcher{
		adapters: make(map[string]adapter.Adapter),
		logger:   logger,
	}
}

// Register adds an adapter to the dispatcher.
func (d *Dispatcher) Register(a adapter.Adapter) {
	d.adapters[a.Type()] = a
	d.logger.Info("registered notification adapter", zap.String("type", a.Type()))
}

// Send dispatches a message to all provided channels. A single channel failure
// does not affect delivery to other channels.
func (d *Dispatcher) Send(
	ctx context.Context,
	channels []model.NotificationChannel,
	msg adapter.Message,
) []SendResult {
	results := make([]SendResult, 0, len(channels))

	for i := range channels {
		ch := &channels[i]
		if !ch.Enabled {
			continue
		}

		a, ok := d.adapters[ch.ChannelType]
		if !ok {
			d.logger.Warn("no adapter for channel type",
				zap.String("type", ch.ChannelType),
				zap.Int64("channel_id", ch.ChannelID),
			)
			results = append(results, SendResult{
				ChannelType: ch.ChannelType,
				Success:     false,
				Error:       ErrAdapterNotFound,
			})
			continue
		}

		err := a.Send(ctx, ch.Config, msg)
		if err != nil {
			d.logger.Error("notification send failed",
				zap.String("type", ch.ChannelType),
				zap.Int64("channel_id", ch.ChannelID),
				zap.Error(err),
			)
			results = append(results, SendResult{
				ChannelType: ch.ChannelType,
				Success:     false,
				Error:       err,
			})
			continue
		}

		d.logger.Info("notification sent",
			zap.String("type", ch.ChannelType),
			zap.Int64("channel_id", ch.ChannelID),
		)
		results = append(results, SendResult{
			ChannelType: ch.ChannelType,
			Success:     true,
		})
	}

	return results
}
