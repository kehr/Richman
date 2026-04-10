// Package schedule provides CRUD operations for user schedule settings and
// holding-level overrides, together with next-analysis-time computation.
package schedule

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"go.uber.org/zap"

	"github.com/richman/backend/internal/model"
)

// reHHMM matches exactly "HH:MM" with two-digit hour and minute.
var reHHMM = regexp.MustCompile(`^\d{2}:\d{2}$`)

// validFrequencies is the complete set of allowed frequency values for global
// and per-market frequency fields.
var validFrequencies = map[string]struct{}{
	model.FrequencyEveryWindow: {},
	model.FrequencyDaily:       {},
	model.FrequencyEvery2Days:  {},
	model.FrequencyEvery3Days:  {},
	model.FrequencyWeekly:      {},
	model.FrequencyCustom:      {},
}

// validWindows is the complete set of allowed window values for holding overrides.
var validWindows = map[string]struct{}{
	model.WindowPre:  {},
	model.WindowPost: {},
	model.WindowBoth: {},
}

// ScheduleRepo is the subset of repo.ScheduleRepo that this service depends on.
// The interface enables unit testing without a live database.
type ScheduleRepo interface {
	GetUserScheduleSettings(
		ctx context.Context, userID int64,
	) (*model.UserScheduleSettings, error)
	UpsertUserScheduleSettings(
		ctx context.Context, userID int64, in *model.UpsertScheduleSettingsInput,
	) (*model.UserScheduleSettings, error)
	ListActiveUserScheduleSettings(ctx context.Context) ([]model.UserScheduleSettings, error)
	GetHoldingScheduleOverride(
		ctx context.Context, userID, holdingID int64,
	) (*model.HoldingScheduleOverride, error)
	UpsertHoldingScheduleOverride(
		ctx context.Context, userID, holdingID int64, in *model.UpsertHoldingScheduleOverrideInput,
	) (*model.HoldingScheduleOverride, error)
}

// Service is the schedule service. It enforces validation rules at the service
// boundary and delegates persistence to ScheduleRepo.
type Service struct {
	repo   ScheduleRepo
	logger *zap.Logger
}

// NewService constructs a new schedule Service.
func NewService(repo ScheduleRepo, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// GetUserScheduleSettings returns the user's active schedule settings. When no
// row exists in the database the system defaults are returned with UserID filled
// in so the caller can distinguish a default from an empty result.
func (s *Service) GetUserScheduleSettings(
	ctx context.Context, userID int64,
) (*model.UserScheduleSettings, error) {
	settings, err := s.repo.GetUserScheduleSettings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user schedule settings: %w", err)
	}
	if settings == nil {
		s.logger.Debug("no schedule settings found, returning defaults", zap.Int64("user_id", userID))
		d := DefaultScheduleSettings()
		d.UserID = userID
		return d, nil
	}
	return settings, nil
}

// UpsertUserScheduleSettings validates the input and then creates or updates
// the user's schedule settings row.
func (s *Service) UpsertUserScheduleSettings(
	ctx context.Context, userID int64, input *model.UpsertScheduleSettingsInput,
) (*model.UserScheduleSettings, error) {
	if err := validateScheduleSettingsInput(input); err != nil {
		return nil, err
	}
	result, err := s.repo.UpsertUserScheduleSettings(ctx, userID, input)
	if err != nil {
		return nil, fmt.Errorf("upsert user schedule settings: %w", err)
	}
	s.logger.Info("upserted user schedule settings", zap.Int64("user_id", userID))
	return result, nil
}

// GetHoldingScheduleOverride returns the active override for a (user, holding)
// pair. Returns nil, nil when no override has been saved — this is a valid state
// meaning "follow the market default".
func (s *Service) GetHoldingScheduleOverride(
	ctx context.Context, userID, holdingID int64,
) (*model.HoldingScheduleOverride, error) {
	override, err := s.repo.GetHoldingScheduleOverride(ctx, userID, holdingID)
	if err != nil {
		return nil, fmt.Errorf("get holding schedule override: %w", err)
	}
	return override, nil
}

// UpsertHoldingScheduleOverride validates the input and creates or updates the
// override for a (user, holding) pair.
func (s *Service) UpsertHoldingScheduleOverride(
	ctx context.Context, userID, holdingID int64, input *model.UpsertHoldingScheduleOverrideInput,
) (*model.HoldingScheduleOverride, error) {
	if err := validateHoldingOverrideInput(input); err != nil {
		return nil, err
	}
	result, err := s.repo.UpsertHoldingScheduleOverride(ctx, userID, holdingID, input)
	if err != nil {
		return nil, fmt.Errorf("upsert holding schedule override: %w", err)
	}
	s.logger.Info("upserted holding schedule override",
		zap.Int64("user_id", userID),
		zap.Int64("holding_id", holdingID),
	)
	return result, nil
}

// ListActiveUserScheduleSettings returns all active schedule settings rows.
// Used by the scheduler to load the full set of user configurations on startup.
func (s *Service) ListActiveUserScheduleSettings(
	ctx context.Context,
) ([]model.UserScheduleSettings, error) {
	list, err := s.repo.ListActiveUserScheduleSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active user schedule settings: %w", err)
	}
	return list, nil
}

// validateScheduleSettingsInput enforces all domain rules for the writable
// schedule settings fields. Validation is centralized here so every caller
// (HTTP handler, internal tool, test) receives identical guarantees.
func validateScheduleSettingsInput(in *model.UpsertScheduleSettingsInput) error {
	// --- Global frequency ---
	if _, ok := validFrequencies[in.GlobalFrequency]; !ok {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_FREQUENCY",
			fmt.Sprintf("global_frequency %q is not allowed", in.GlobalFrequency))
	}
	if in.GlobalFrequency == model.FrequencyCustom {
		if in.GlobalFrequencyDays == nil {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_FREQUENCY",
				"global_frequency_days is required when global_frequency is \"custom\"")
		}
		if *in.GlobalFrequencyDays < 1 || *in.GlobalFrequencyDays > 30 {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_FREQUENCY",
				"global_frequency_days must be between 1 and 30")
		}
	} else if in.GlobalFrequencyDays != nil {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_FREQUENCY",
			"global_frequency_days must be null when global_frequency is not \"custom\"")
	}

	// --- A-share window times ---
	if err := validateWindowTime(in.ASharePreTime, "a_share_pre_time", 7*60, 9*60+29); err != nil {
		return err
	}
	if err := validateWindowTime(in.ASharePostTime, "a_share_post_time", 15*60, 20*60); err != nil {
		return err
	}

	// --- US window times ---
	if err := validateWindowTime(in.USPreTime, "us_pre_time", 20*60, 23*60); err != nil {
		return err
	}
	if err := validateWindowTime(in.USPostTime, "us_post_time", 4*60, 8*60); err != nil {
		return err
	}

	// --- Per-market frequencies (optional / nullable) ---
	if in.AShareFrequency != nil {
		if err := validateMarketFrequency(*in.AShareFrequency, in.AShareFrequencyDays, "a_share"); err != nil {
			return err
		}
	} else if in.AShareFrequencyDays != nil {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_FREQUENCY",
			"a_share_frequency_days must be null when a_share_frequency is null")
	}

	if in.USFrequency != nil {
		if err := validateMarketFrequency(*in.USFrequency, in.USFrequencyDays, "us"); err != nil {
			return err
		}
	} else if in.USFrequencyDays != nil {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_FREQUENCY",
			"us_frequency_days must be null when us_frequency is null")
	}

	return nil
}

// validateHoldingOverrideInput enforces domain rules for holding schedule
// override fields.
func validateHoldingOverrideInput(in *model.UpsertHoldingScheduleOverrideInput) error {
	// frequency: null means "follow market", otherwise must be a valid value.
	if in.Frequency != nil && *in.Frequency != "" {
		if _, ok := validFrequencies[*in.Frequency]; !ok {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_FREQUENCY",
				fmt.Sprintf("frequency %q is not allowed", *in.Frequency))
		}
		if *in.Frequency == model.FrequencyCustom {
			if in.FrequencyDays == nil {
				return model.NewAppError(http.StatusBadRequest,
					"INVALID_FREQUENCY",
					"frequency_days is required when frequency is \"custom\"")
			}
			if *in.FrequencyDays < 1 || *in.FrequencyDays > 30 {
				return model.NewAppError(http.StatusBadRequest,
					"INVALID_FREQUENCY",
					"frequency_days must be between 1 and 30")
			}
		} else if in.FrequencyDays != nil {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_FREQUENCY",
				"frequency_days must be null when frequency is not \"custom\"")
		}
	} else if in.FrequencyDays != nil {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_FREQUENCY",
			"frequency_days must be null when frequency is null")
	}

	// window: null/empty means "follow market", otherwise must be pre/post/both.
	if in.Window != nil && *in.Window != "" {
		if _, ok := validWindows[*in.Window]; !ok {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_WINDOW",
				fmt.Sprintf("window %q is not allowed; must be pre, post, or both", *in.Window))
		}
	}

	return nil
}

// validateMarketFrequency validates a per-market frequency field together with
// its optional custom-days companion. The market parameter is used only for
// error message clarity.
func validateMarketFrequency(freq string, days *int32, market string) error {
	if _, ok := validFrequencies[freq]; !ok {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_FREQUENCY",
			fmt.Sprintf("%s_frequency %q is not allowed", market, freq))
	}
	if freq == model.FrequencyCustom {
		if days == nil {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_FREQUENCY",
				fmt.Sprintf("%s_frequency_days is required when %s_frequency is \"custom\"", market, market))
		}
		if *days < 1 || *days > 30 {
			return model.NewAppError(http.StatusBadRequest,
				"INVALID_FREQUENCY",
				fmt.Sprintf("%s_frequency_days must be between 1 and 30", market))
		}
	} else if days != nil {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_FREQUENCY",
			fmt.Sprintf("%s_frequency_days must be null when %s_frequency is not \"custom\"", market, market))
	}
	return nil
}

// validateWindowTime checks that t is in "HH:MM" format, that the minute is
// divisible by 5, and that the time falls within [minMinutes, maxMinutes]
// inclusive (expressed as total minutes since midnight).
func validateWindowTime(t, field string, minMinutes, maxMinutes int) error {
	if !reHHMM.MatchString(t) {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_TIME",
			fmt.Sprintf("%s must be in HH:MM format", field))
	}
	hour, _ := strconv.Atoi(t[:2])
	minute, _ := strconv.Atoi(t[3:])
	if hour > 23 || minute > 59 {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_TIME",
			fmt.Sprintf("%s has an invalid hour or minute value", field))
	}
	if minute%5 != 0 {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_TIME",
			fmt.Sprintf("%s minutes must be divisible by 5", field))
	}
	total := hour*60 + minute
	if total < minMinutes || total > maxMinutes {
		return model.NewAppError(http.StatusBadRequest,
			"INVALID_TIME",
			fmt.Sprintf("%s must be between %02d:%02d and %02d:%02d",
				field,
				minMinutes/60, minMinutes%60,
				maxMinutes/60, maxMinutes%60,
			))
	}
	return nil
}
