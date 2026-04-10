package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// ScheduleRepo handles data access for user_schedule_settings and
// holding_schedule_overrides tables.
type ScheduleRepo struct {
	pool *pgxpool.Pool
}

// NewScheduleRepo creates a new ScheduleRepo backed by the given pgx pool.
func NewScheduleRepo(pool *pgxpool.Pool) *ScheduleRepo {
	return &ScheduleRepo{pool: pool}
}

// scheduleSettingsColumns is the canonical column list for SELECT queries on
// user_schedule_settings. TIME columns are cast to TEXT for direct string
// scanning, avoiding a pgtype.Time dependency. Order must match scanScheduleSettings.
const scheduleSettingsColumns = `id, user_id,
	global_frequency, global_frequency_days,
	a_share_pre_enabled, a_share_pre_time::text, a_share_pre_custom,
	a_share_post_enabled, a_share_post_time::text, a_share_post_custom,
	a_share_frequency, a_share_frequency_days,
	us_pre_enabled, us_pre_time::text, us_pre_custom,
	us_post_enabled, us_post_time::text, us_post_custom,
	us_frequency, us_frequency_days,
	created_at, updated_at, creator, modifier, is_deleted`

// scanScheduleSettings reads the canonical user_schedule_settings columns into
// a model.UserScheduleSettings. Argument order must match scheduleSettingsColumns.
func scanScheduleSettings(row pgx.Row, s *model.UserScheduleSettings) error {
	var (
		globalFrequencyDays sql.NullInt32
		aShareFrequency     sql.NullString
		aShareFrequencyDays sql.NullInt32
		usFrequency         sql.NullString
		usFrequencyDays     sql.NullInt32
	)
	err := row.Scan(
		&s.ID, &s.UserID,
		&s.GlobalFrequency, &globalFrequencyDays,
		&s.ASharePreEnabled, &s.ASharePreTime, &s.ASharePreCustom,
		&s.ASharePostEnabled, &s.ASharePostTime, &s.ASharePostCustom,
		&aShareFrequency, &aShareFrequencyDays,
		&s.USPreEnabled, &s.USPreTime, &s.USPreCustom,
		&s.USPostEnabled, &s.USPostTime, &s.USPostCustom,
		&usFrequency, &usFrequencyDays,
		&s.CreatedAt, &s.UpdatedAt, &s.Creator, &s.Modifier, &s.IsDeleted,
	)
	if err != nil {
		return err
	}
	if globalFrequencyDays.Valid {
		v := globalFrequencyDays.Int32
		s.GlobalFrequencyDays = &v
	}
	if aShareFrequency.Valid {
		v := aShareFrequency.String
		s.AShareFrequency = &v
	}
	if aShareFrequencyDays.Valid {
		v := aShareFrequencyDays.Int32
		s.AShareFrequencyDays = &v
	}
	if usFrequency.Valid {
		v := usFrequency.String
		s.USFrequency = &v
	}
	if usFrequencyDays.Valid {
		v := usFrequencyDays.Int32
		s.USFrequencyDays = &v
	}
	return nil
}

// GetUserScheduleSettings returns the single active (is_deleted = 0) schedule
// settings row for a user. Returns nil, nil when no record exists (caller
// should apply system defaults).
func (r *ScheduleRepo) GetUserScheduleSettings(
	ctx context.Context, userID int64,
) (*model.UserScheduleSettings, error) {
	var s model.UserScheduleSettings
	row := r.pool.QueryRow(ctx,
		`SELECT `+scheduleSettingsColumns+`
		 FROM user_schedule_settings
		 WHERE user_id = $1 AND is_deleted = 0`,
		userID,
	)
	if err := scanScheduleSettings(row, &s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user schedule settings: %w", err)
	}
	return &s, nil
}

// UpsertUserScheduleSettings creates or updates the active schedule settings
// for a user. Uses ON CONFLICT targeting the partial unique index
// uq_user_schedule_settings_active_user (WHERE is_deleted = 0). Returns the
// fully populated row after insert/update.
func (r *ScheduleRepo) UpsertUserScheduleSettings(
	ctx context.Context, userID int64, in *model.UpsertScheduleSettingsInput,
) (*model.UserScheduleSettings, error) {
	var s model.UserScheduleSettings
	row := r.pool.QueryRow(ctx,
		`INSERT INTO user_schedule_settings (
			user_id,
			global_frequency, global_frequency_days,
			a_share_pre_enabled, a_share_pre_time, a_share_pre_custom,
			a_share_post_enabled, a_share_post_time, a_share_post_custom,
			a_share_frequency, a_share_frequency_days,
			us_pre_enabled, us_pre_time, us_pre_custom,
			us_post_enabled, us_post_time, us_post_custom,
			us_frequency, us_frequency_days,
			creator, modifier
		) VALUES (
			$1,
			$2, $3,
			$4, $5::time, $6,
			$7, $8::time, $9,
			$10, $11,
			$12, $13::time, $14,
			$15, $16::time, $17,
			$18, $19,
			$20, $20
		)
		ON CONFLICT (user_id) WHERE is_deleted = 0
		DO UPDATE SET
			global_frequency      = EXCLUDED.global_frequency,
			global_frequency_days = EXCLUDED.global_frequency_days,
			a_share_pre_enabled   = EXCLUDED.a_share_pre_enabled,
			a_share_pre_time      = EXCLUDED.a_share_pre_time,
			a_share_pre_custom    = EXCLUDED.a_share_pre_custom,
			a_share_post_enabled  = EXCLUDED.a_share_post_enabled,
			a_share_post_time     = EXCLUDED.a_share_post_time,
			a_share_post_custom   = EXCLUDED.a_share_post_custom,
			a_share_frequency     = EXCLUDED.a_share_frequency,
			a_share_frequency_days = EXCLUDED.a_share_frequency_days,
			us_pre_enabled        = EXCLUDED.us_pre_enabled,
			us_pre_time           = EXCLUDED.us_pre_time,
			us_pre_custom         = EXCLUDED.us_pre_custom,
			us_post_enabled       = EXCLUDED.us_post_enabled,
			us_post_time          = EXCLUDED.us_post_time,
			us_post_custom        = EXCLUDED.us_post_custom,
			us_frequency          = EXCLUDED.us_frequency,
			us_frequency_days     = EXCLUDED.us_frequency_days,
			modifier              = EXCLUDED.modifier,
			updated_at            = now()
		RETURNING `+scheduleSettingsColumns,
		userID,
		in.GlobalFrequency, in.GlobalFrequencyDays,
		in.ASharePreEnabled, in.ASharePreTime, in.ASharePreCustom,
		in.ASharePostEnabled, in.ASharePostTime, in.ASharePostCustom,
		in.AShareFrequency, in.AShareFrequencyDays,
		in.USPreEnabled, in.USPreTime, in.USPreCustom,
		in.USPostEnabled, in.USPostTime, in.USPostCustom,
		in.USFrequency, in.USFrequencyDays,
		in.Modifier,
	)
	if err := scanScheduleSettings(row, &s); err != nil {
		return nil, fmt.Errorf("upsert user schedule settings: %w", err)
	}
	return &s, nil
}

// ListActiveUserScheduleSettings returns all active (is_deleted = 0) schedule
// settings rows. Used by the scheduler on startup to load all user configs.
func (r *ScheduleRepo) ListActiveUserScheduleSettings(
	ctx context.Context,
) ([]model.UserScheduleSettings, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scheduleSettingsColumns+`
		 FROM user_schedule_settings
		 WHERE is_deleted = 0
		 ORDER BY user_id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active user schedule settings: %w", err)
	}
	defer rows.Close()

	var result []model.UserScheduleSettings
	for rows.Next() {
		var s model.UserScheduleSettings
		if err := scanScheduleSettings(rows, &s); err != nil {
			return nil, fmt.Errorf("scan user schedule settings: %w", err)
		}
		result = append(result, s)
	}
	return result, nil
}

// holdingOverrideColumns is the canonical column list for SELECT queries on
// holding_schedule_overrides. "window" is a reserved word and must be quoted.
// Order must match scanHoldingOverride.
const holdingOverrideColumns = `id, user_id, holding_id,
	frequency, frequency_days, "window",
	created_at, updated_at, creator, modifier, is_deleted`

// scanHoldingOverride reads the canonical holding_schedule_overrides columns
// into a model.HoldingScheduleOverride. Argument order must match
// holdingOverrideColumns.
func scanHoldingOverride(row pgx.Row, h *model.HoldingScheduleOverride) error {
	var (
		frequency     sql.NullString
		frequencyDays sql.NullInt32
		window        sql.NullString
	)
	err := row.Scan(
		&h.ID, &h.UserID, &h.HoldingID,
		&frequency, &frequencyDays, &window,
		&h.CreatedAt, &h.UpdatedAt, &h.Creator, &h.Modifier, &h.IsDeleted,
	)
	if err != nil {
		return err
	}
	if frequency.Valid {
		v := frequency.String
		h.Frequency = &v
	}
	if frequencyDays.Valid {
		v := frequencyDays.Int32
		h.FrequencyDays = &v
	}
	if window.Valid {
		v := window.String
		h.Window = &v
	}
	return nil
}

// GetHoldingScheduleOverride returns the single active (is_deleted = 0) holding
// schedule override for a (user, holding) pair. Returns nil, nil when no record
// exists.
func (r *ScheduleRepo) GetHoldingScheduleOverride(
	ctx context.Context, userID, holdingID int64,
) (*model.HoldingScheduleOverride, error) {
	var h model.HoldingScheduleOverride
	row := r.pool.QueryRow(ctx,
		`SELECT `+holdingOverrideColumns+`
		 FROM holding_schedule_overrides
		 WHERE user_id = $1 AND holding_id = $2 AND is_deleted = 0`,
		userID, holdingID,
	)
	if err := scanHoldingOverride(row, &h); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query holding schedule override: %w", err)
	}
	return &h, nil
}

// UpsertHoldingScheduleOverride creates or updates the active override for a
// (user, holding) pair. Uses ON CONFLICT targeting the partial unique index
// uq_holding_schedule_overrides_active (WHERE is_deleted = 0). Returns the
// fully populated row after insert/update.
func (r *ScheduleRepo) UpsertHoldingScheduleOverride(
	ctx context.Context, userID, holdingID int64, in *model.UpsertHoldingScheduleOverrideInput,
) (*model.HoldingScheduleOverride, error) {
	var h model.HoldingScheduleOverride
	row := r.pool.QueryRow(ctx,
		`INSERT INTO holding_schedule_overrides (
			user_id, holding_id,
			frequency, frequency_days, "window",
			creator, modifier
		) VALUES (
			$1, $2,
			$3, $4, $5,
			$6, $6
		)
		ON CONFLICT (user_id, holding_id) WHERE is_deleted = 0
		DO UPDATE SET
			frequency      = EXCLUDED.frequency,
			frequency_days = EXCLUDED.frequency_days,
			"window"       = EXCLUDED."window",
			modifier       = EXCLUDED.modifier,
			updated_at     = now()
		RETURNING `+holdingOverrideColumns,
		userID, holdingID,
		in.Frequency, in.FrequencyDays, in.Window,
		in.Modifier,
	)
	if err := scanHoldingOverride(row, &h); err != nil {
		return nil, fmt.Errorf("upsert holding schedule override: %w", err)
	}
	return &h, nil
}
