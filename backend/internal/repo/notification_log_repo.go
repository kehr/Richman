package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// NotificationLogRepo handles notification log data access operations.
type NotificationLogRepo struct {
	pool *pgxpool.Pool
}

// NewNotificationLogRepo creates a new NotificationLogRepo.
func NewNotificationLogRepo(pool *pgxpool.Pool) *NotificationLogRepo {
	return &NotificationLogRepo{pool: pool}
}

// Create inserts a new notification log entry.
func (r *NotificationLogRepo) Create(
	ctx context.Context,
	userID int64,
	channelType, messageType, status, errorMessage string,
) (*model.NotificationLog, error) {
	var log model.NotificationLog
	err := r.pool.QueryRow(ctx,
		`INSERT INTO notification_logs
		 (user_id, channel_type, message_type, status, error_message, creator, modifier)
		 VALUES ($1, $2, $3, $4, $5, 'system', 'system')
		 RETURNING notification_log_id, user_id, channel_type, message_type,
		           status, error_message, created_at`,
		userID, channelType, messageType, status, errorMessage,
	).Scan(
		&log.LogID, &log.UserID, &log.ChannelType, &log.MessageType,
		&log.Status, &log.ErrorMessage, &log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert notification log: %w", err)
	}
	return &log, nil
}
