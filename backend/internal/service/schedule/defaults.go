package schedule

import "github.com/richman/backend/internal/model"

// System-default schedule settings. These are applied when a user has not yet
// saved any custom schedule configuration. Values follow the product spec:
//   - A-share pre-market:  08:30 (enabled)
//   - A-share post-market: 15:05 (enabled)
//   - US pre-market:       20:30 Asia/Shanghai ≈ EDT 08:30 (disabled by default)
//   - US post-market:      04:05 Asia/Shanghai ≈ EDT 16:05 (enabled)
//   - Global frequency:    daily
//
// EDT = Eastern Daylight Time (summer, UTC-4); NYSE open 09:30 EDT = 21:30+8, close 16:00 EDT = 04:00+8.
// EST = Eastern Standard Time (winter, UTC-5); NYSE open 09:30 EST = 22:30+8, close 16:00 EST = 05:00+8.
// The +5-minute offset on post-market times allows the closing print to settle before triggering analysis.
const (
	defaultGlobalFrequency = model.FrequencyDaily

	defaultASharePreEnabled  = true
	defaultASharePreTime     = "08:30"
	defaultASharePreCustom   = false
	defaultASharePostEnabled = true
	defaultASharePostTime    = "15:05"
	defaultASharePostCustom  = false

	// EDT (summer) US window defaults.
	defaultUSPreEnabled  = false
	defaultUSPreTime     = "20:30" // NYSE 09:30 EDT = 21:30+8; stored as Asia/Shanghai
	defaultUSPreCustom   = false
	defaultUSPostEnabled = true
	defaultUSPostTime    = "04:05" // NYSE 16:00 EDT + 5min = 04:05+8
	defaultUSPostCustom  = false

	// EST (winter) US window defaults. Exported for use by the DST logic layer
	// (Step 4) which selects between EDT and EST windows based on the current
	// US clock change date.
	DefaultUSPreTimeEST  = "21:30" // NYSE 09:30 EST = 22:30+8; stored as Asia/Shanghai
	DefaultUSPostTimeEST = "05:05" // NYSE 16:00 EST + 5min = 05:05+8
)

// DefaultScheduleSettings returns a UserScheduleSettings populated with
// system defaults. id and user_id are set to 0; the caller should set
// UserID before returning to the client.
func DefaultScheduleSettings() *model.UserScheduleSettings {
	return &model.UserScheduleSettings{
		ID:     0,
		UserID: 0,

		GlobalFrequency: defaultGlobalFrequency,

		ASharePreEnabled:  defaultASharePreEnabled,
		ASharePreTime:     defaultASharePreTime,
		ASharePreCustom:   defaultASharePreCustom,
		ASharePostEnabled: defaultASharePostEnabled,
		ASharePostTime:    defaultASharePostTime,
		ASharePostCustom:  defaultASharePostCustom,

		USPreEnabled:  defaultUSPreEnabled,
		USPreTime:     defaultUSPreTime,
		USPreCustom:   defaultUSPreCustom,
		USPostEnabled: defaultUSPostEnabled,
		USPostTime:    defaultUSPostTime,
		USPostCustom:  defaultUSPostCustom,
	}
}
