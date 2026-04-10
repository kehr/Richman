package schedule

import "github.com/richman/backend/internal/model"

// System-default schedule settings. These are applied when a user has not yet
// saved any custom schedule configuration. Values follow the product spec:
//   - A-share pre-market:  08:30 (enabled)
//   - A-share post-market: 15:05 (enabled)
//   - US pre-market:       20:30 Asia/Shanghai ≈ EDT 08:30 (disabled by default)
//   - US post-market:      04:05 Asia/Shanghai ≈ EDT 16:05 (enabled)
//   - Global frequency:    daily
const (
	defaultGlobalFrequency = model.FrequencyDaily

	defaultASharePreEnabled  = true
	defaultASharePreTime     = "08:30"
	defaultASharePreCustom   = false
	defaultASharePostEnabled = true
	defaultASharePostTime    = "15:05"
	defaultASharePostCustom  = false

	defaultUSPreEnabled  = false
	defaultUSPreTime     = "20:30"
	defaultUSPreCustom   = false
	defaultUSPostEnabled = true
	defaultUSPostTime    = "04:05"
	defaultUSPostCustom  = false
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
