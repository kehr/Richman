package repo

import (
	"context"
	"fmt"
	"time"

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

// CountTodayByUser returns the number of notification log entries for a user
// on today's calendar date (UTC) filtered by channelType. Used by the email
// push service to enforce the daily per-user push frequency cap.
func (r *NotificationLogRepo) CountTodayByUser(ctx context.Context, userID int64, channelType string) (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM rm_notification_logs
		 WHERE user_id = $1
		   AND channel_type = $2
		   AND created_at >= $3
		   AND created_at < $4`,
		userID, channelType, today, tomorrow,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count today notification logs: %w", err)
	}
	return count, nil
}

// Create inserts a new notification log entry.
func (r *NotificationLogRepo) Create(
	ctx context.Context,
	userID int64,
	channelType, messageType, status, errorMessage string,
) (*model.NotificationLog, error) {
	var log model.NotificationLog
	err := r.pool.QueryRow(ctx,
		`INSERT INTO rm_notification_logs
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
