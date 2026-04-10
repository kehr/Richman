package model

import "time"

// Frequency values for analysis scheduling. These must stay in sync with the
// CHECK constraint (or application-level validation) on the global_frequency,
// a_share_frequency, and us_frequency columns.
const (
	FrequencyEveryWindow = "every_window"
	FrequencyDaily       = "daily"
	FrequencyEvery2Days  = "every_2_days"
	FrequencyEvery3Days  = "every_3_days"
	FrequencyWeekly      = "weekly"
	FrequencyCustom      = "custom"
)

// WindowValue values for holding_schedule_overrides."window" column.
const (
	WindowPre  = "pre"
	WindowPost = "post"
	WindowBoth = "both"
)

// UserScheduleSettings stores per-user global and per-market analysis schedule
// preferences. At most one active (is_deleted = 0) row exists per user,
// enforced by the partial unique index uq_user_schedule_settings_active_user.
type UserScheduleSettings struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"userId"`

	// Global frequency
	GlobalFrequency     string `json:"globalFrequency"`
	GlobalFrequencyDays *int32 `json:"globalFrequencyDays,omitempty"`

	// A-share windows
	ASharePreEnabled    bool    `json:"aSharePreEnabled"`
	ASharePreTime       string  `json:"aSharePreTime"` // stored as HH:MM
	ASharePreCustom     bool    `json:"aSharePreCustom"`
	ASharePostEnabled   bool    `json:"aSharePostEnabled"`
	ASharePostTime      string  `json:"aSharePostTime"` // stored as HH:MM
	ASharePostCustom    bool    `json:"aSharePostCustom"`
	AShareFrequency     *string `json:"aShareFrequency,omitempty"`
	AShareFrequencyDays *int32  `json:"aShareFrequencyDays,omitempty"`

	// US stock / gold windows (times stored in Asia/Shanghai)
	USPreEnabled    bool    `json:"usPreEnabled"`
	USPreTime       string  `json:"usPreTime"` // stored as HH:MM
	USPreCustom     bool    `json:"usPreCustom"`
	USPostEnabled   bool    `json:"usPostEnabled"`
	USPostTime      string  `json:"usPostTime"` // stored as HH:MM
	USPostCustom    bool    `json:"usPostCustom"`
	USFrequency     *string `json:"usFrequency,omitempty"`
	USFrequencyDays *int32  `json:"usFrequencyDays,omitempty"`

	// Audit fields
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Creator   string    `json:"creator"`
	Modifier  string    `json:"modifier"`
	IsDeleted int16     `json:"-"`
}

// HoldingScheduleOverride stores per-holding analysis schedule overrides for a
// user. Null fields mean "follow the market-level default from
// UserScheduleSettings". At most one active (is_deleted = 0) row exists per
// (user_id, holding_id) pair, enforced by the partial unique index
// uq_holding_schedule_overrides_active.
type HoldingScheduleOverride struct {
	ID        int64 `json:"id"`
	UserID    int64 `json:"userId"`
	HoldingID int64 `json:"holdingId"`

	Frequency     *string `json:"frequency,omitempty"`
	FrequencyDays *int32  `json:"frequencyDays,omitempty"`
	Window        *string `json:"window,omitempty"` // pre | post | both | null

	// Audit fields
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Creator   string    `json:"creator"`
	Modifier  string    `json:"modifier"`
	IsDeleted int16     `json:"-"`
}

// UpsertScheduleSettingsInput carries the full set of user-writable fields for
// UpsertUserScheduleSettings. All fields are required on write; null frequency
// overrides are represented by nil pointers.
type UpsertScheduleSettingsInput struct {
	GlobalFrequency     string
	GlobalFrequencyDays *int32

	ASharePreEnabled    bool
	ASharePreTime       string
	ASharePreCustom     bool
	ASharePostEnabled   bool
	ASharePostTime      string
	ASharePostCustom    bool
	AShareFrequency     *string
	AShareFrequencyDays *int32

	USPreEnabled    bool
	USPreTime       string
	USPreCustom     bool
	USPostEnabled   bool
	USPostTime      string
	USPostCustom    bool
	USFrequency     *string
	USFrequencyDays *int32

	Modifier string
}

// UpsertHoldingScheduleOverrideInput carries the user-writable fields for
// UpsertHoldingScheduleOverride. Nil pointers mean "follow market default".
type UpsertHoldingScheduleOverrideInput struct {
	Frequency     *string
	FrequencyDays *int32
	Window        *string
	Modifier      string
}
