package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// NotificationChannelRepo handles notification channel data access operations.
type NotificationChannelRepo struct {
	pool *pgxpool.Pool
}

// NewNotificationChannelRepo creates a new NotificationChannelRepo.
func NewNotificationChannelRepo(pool *pgxpool.Pool) *NotificationChannelRepo {
	return &NotificationChannelRepo{pool: pool}
}

// ListByUser returns all active notification channels for a user.
func (r *NotificationChannelRepo) ListByUser(ctx context.Context, userID int64) ([]model.NotificationChannel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT notification_channel_id, user_id, channel_type, config, enabled,
		        created_at, updated_at
		 FROM rm_notification_channels
		 WHERE user_id = $1 AND is_deleted = 0
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query notification channels: %w", err)
	}
	defer rows.Close()

	var channels []model.NotificationChannel
	for rows.Next() {
		var ch model.NotificationChannel
		var enabledSmall int16
		if err := rows.Scan(
			&ch.ChannelID, &ch.UserID, &ch.ChannelType, &ch.Config, &enabledSmall,
			&ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification channel: %w", err)
		}
		ch.Enabled = enabledSmall == 1
		channels = append(channels, ch)
	}
	return channels, nil
}

// GetByID returns a single notification channel by ID. Returns nil if not found.
func (r *NotificationChannelRepo) GetByID(ctx context.Context, channelID int64) (*model.NotificationChannel, error) {
	var ch model.NotificationChannel
	var enabledSmall int16
	err := r.pool.QueryRow(ctx,
		`SELECT notification_channel_id, user_id, channel_type, config, enabled,
		        created_at, updated_at
		 FROM rm_notification_channels
		 WHERE notification_channel_id = $1 AND is_deleted = 0`,
		channelID,
	).Scan(
		&ch.ChannelID, &ch.UserID, &ch.ChannelType, &ch.Config, &enabledSmall,
		&ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query notification channel by id: %w", err)
	}
	ch.Enabled = enabledSmall == 1
	return &ch, nil
}

// Create inserts a new notification channel and returns it.
func (r *NotificationChannelRepo) Create(
	ctx context.Context, userID int64,
	input *model.CreateChannelInput, creator string,
) (*model.NotificationChannel, error) {
	var ch model.NotificationChannel
	var enabledSmall int16
	err := r.pool.QueryRow(ctx,
		`INSERT INTO rm_notification_channels
		 (user_id, channel_type, config, creator, modifier)
		 VALUES ($1, $2, $3, $4, $4)
		 RETURNING notification_channel_id, user_id, channel_type, config, enabled,
		           created_at, updated_at`,
		userID, input.ChannelType, input.Config, creator,
	).Scan(
		&ch.ChannelID, &ch.UserID, &ch.ChannelType, &ch.Config, &enabledSmall,
		&ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert notification channel: %w", err)
	}
	ch.Enabled = enabledSmall == 1
	return &ch, nil
}

// Update updates specified fields on a notification channel and returns the updated row.
func (r *NotificationChannelRepo) Update(
	ctx context.Context, channelID int64,
	input *model.UpdateChannelInput, modifier string,
) (*model.NotificationChannel, error) {
	// Convert *bool to *int16 for DB storage.
	var enabledVal *int16
	if input.Enabled != nil {
		v := int16(0)
		if *input.Enabled {
			v = 1
		}
		enabledVal = &v
	}

	var ch model.NotificationChannel
	var enabledSmall int16
	err := r.pool.QueryRow(ctx,
		`UPDATE rm_notification_channels SET
			config = COALESCE($1, config),
			enabled = COALESCE($2, enabled),
			modifier = $3,
			updated_at = NOW()
		 WHERE notification_channel_id = $4 AND is_deleted = 0
		 RETURNING notification_channel_id, user_id, channel_type, config, enabled,
		           created_at, updated_at`,
		input.Config, enabledVal, modifier, channelID,
	).Scan(
		&ch.ChannelID, &ch.UserID, &ch.ChannelType, &ch.Config, &enabledSmall,
		&ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update notification channel: %w", err)
	}
	ch.Enabled = enabledSmall == 1
	return &ch, nil
}

// SoftDelete marks a notification channel as deleted.
func (r *NotificationChannelRepo) SoftDelete(ctx context.Context, channelID int64, modifier string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE rm_notification_channels SET is_deleted = 1, modifier = $1, updated_at = NOW()
		 WHERE notification_channel_id = $2 AND is_deleted = 0`,
		modifier, channelID,
	)
	if err != nil {
		return fmt.Errorf("soft delete notification channel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("notification channel not found")
	}
	return nil
}

// ListEnabledByUserIDs returns all enabled notification channels for a list of user IDs.
func (r *NotificationChannelRepo) ListEnabledByUserIDs(
	ctx context.Context, userIDs []int64,
) ([]model.NotificationChannel, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT notification_channel_id, user_id, channel_type, config, enabled,
		        created_at, updated_at
		 FROM rm_notification_channels
		 WHERE user_id = ANY($1) AND enabled = 1 AND is_deleted = 0`,
		userIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query enabled channels by user ids: %w", err)
	}
	defer rows.Close()

	var channels []model.NotificationChannel
	for rows.Next() {
		var ch model.NotificationChannel
		var enabledSmall int16
		if err := rows.Scan(
			&ch.ChannelID, &ch.UserID, &ch.ChannelType, &ch.Config, &enabledSmall,
			&ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification channel: %w", err)
		}
		ch.Enabled = enabledSmall == 1
		channels = append(channels, ch)
	}
	return channels, nil
}
