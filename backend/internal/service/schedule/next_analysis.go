package schedule

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/richman/backend/internal/model"
)

// Market identifiers passed to ComputeNextAnalysisAt.
const (
	MarketAShare  = "a_share"
	MarketUSStock = "us_stock"
)

// frequencyToMinHours returns the minimum number of hours that must pass
// between two analysis runs at the given frequency. FrequencyEveryWindow
// returns 0 meaning "no minimum gap; fire on the next enabled window".
// FrequencyCustom requires frequencyDays to be non-nil.
func frequencyToMinHours(freq string, frequencyDays *int32) (int, error) {
	switch freq {
	case model.FrequencyEveryWindow:
		return 0, nil
	case model.FrequencyDaily:
		return 24, nil
	case model.FrequencyEvery2Days:
		return 48, nil
	case model.FrequencyEvery3Days:
		return 72, nil
	case model.FrequencyWeekly:
		return 168, nil
	case model.FrequencyCustom:
		if frequencyDays == nil {
			return 0, fmt.Errorf("frequency_days is required for custom frequency")
		}
		return int(*frequencyDays) * 24, nil
	default:
		return 0, fmt.Errorf("unknown frequency %q", freq)
	}
}

// parseHHMM parses a "HH:MM" string into (hour, minute). It does not validate
// range; the service layer is expected to have validated the stored values.
func parseHHMM(s string) (hour, minute int, err error) {
	if len(s) != 5 || s[2] != ':' {
		return 0, 0, fmt.Errorf("invalid time format %q, expected HH:MM", s)
	}
	hour, err = strconv.Atoi(s[:2])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour in %q: %w", s, err)
	}
	minute, err = strconv.Atoi(s[3:])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute in %q: %w", s, err)
	}
	return hour, minute, nil
}

// windowSpec describes a single trading window (pre or post) for a market day.
type windowSpec struct {
	enabled bool
	timeStr string // "HH:MM"
}

// marketWindows collects the window specs from settings that apply to the
// given market. The window override from a holding record is applied before
// calling this helper; activeWindow is the resolved window value ("pre",
// "post", "both", or "" to follow settings).
func marketWindows(
	settings *model.UserScheduleSettings,
	market string,
	activeWindow string, // "" = use both enabled windows from settings
) ([]windowSpec, error) {
	var pre, post windowSpec

	switch market {
	case MarketAShare:
		pre = windowSpec{enabled: settings.ASharePreEnabled, timeStr: settings.ASharePreTime}
		post = windowSpec{enabled: settings.ASharePostEnabled, timeStr: settings.ASharePostTime}
	case MarketUSStock:
		pre = windowSpec{enabled: settings.USPreEnabled, timeStr: settings.USPreTime}
		post = windowSpec{enabled: settings.USPostEnabled, timeStr: settings.USPostTime}
	default:
		return nil, fmt.Errorf("unknown market %q", market)
	}

	// Apply window override from holding (if any).
	switch activeWindow {
	case model.WindowPre:
		post.enabled = false
	case model.WindowPost:
		pre.enabled = false
	case model.WindowBoth:
		pre.enabled = true
		post.enabled = true
	default:
		// no override — keep the settings-level enabled flags
	}

	var specs []windowSpec
	if pre.enabled {
		specs = append(specs, pre)
	}
	if post.enabled {
		specs = append(specs, post)
	}
	return specs, nil
}

// isWeekend reports whether the given date falls on Saturday or Sunday.
// This is used to skip non-trading days for A-share markets.
func isWeekend(t time.Time) bool {
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// buildCandidatesForDay returns all window time.Time candidates for the given
// calendar day (in the location of the base time). Each window generates one
// candidate. The returned slice is sorted ascending by time.
func buildCandidatesForDay(day time.Time, specs []windowSpec) ([]time.Time, error) {
	var candidates []time.Time
	for _, spec := range specs {
		h, m, err := parseHHMM(spec.timeStr)
		if err != nil {
			return nil, err
		}
		// Construct the window time on this calendar day, preserving the
		// timezone of the base day.
		t := time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, day.Location())
		candidates = append(candidates, t)
	}
	// Sort ascending so we always process pre before post.
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].Before(candidates[j-1]); j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
	return candidates, nil
}

// ComputeNextAnalysisAt determines when the next analysis should be triggered
// for the given (user, holding, market) combination.
//
// Priority for frequency resolution:
//  1. Holding-level override frequency (if set)
//  2. Market-level frequency from UserScheduleSettings (if set)
//  3. Global frequency from UserScheduleSettings
//
// Priority for window resolution:
//  1. Holding-level window override (if set)
//  2. Market-level window enabled flags from UserScheduleSettings
//
// If lastAnalyzedAt is nil the function returns the next upcoming enabled
// window starting from now without applying a minimum interval constraint.
//
// For A-share markets, weekend days are skipped. NYSE holidays are not
// currently filtered (deferred to DST logic layer).
func (s *Service) ComputeNextAnalysisAt(
	ctx context.Context,
	userID, holdingID int64,
	market string,
	lastAnalyzedAt *time.Time,
	now time.Time,
) (time.Time, error) {
	// --- Step 1: Fetch override and settings ---
	override, err := s.GetHoldingScheduleOverride(ctx, userID, holdingID)
	if err != nil {
		return time.Time{}, fmt.Errorf("compute next analysis: get holding override: %w", err)
	}

	settings, err := s.GetUserScheduleSettings(ctx, userID)
	if err != nil {
		return time.Time{}, fmt.Errorf("compute next analysis: get user schedule settings: %w", err)
	}

	// --- Step 2: Resolve frequency (three-level priority) ---
	freq, freqDays := resolveFrequency(override, settings, market)

	minHours, err := frequencyToMinHours(freq, freqDays)
	if err != nil {
		return time.Time{}, fmt.Errorf("compute next analysis: %w", err)
	}

	// --- Step 3: Resolve window override ---
	activeWindow := resolveWindow(override)

	// --- Step 4: Build window specs ---
	specs, err := marketWindows(settings, market, activeWindow)
	if err != nil {
		return time.Time{}, fmt.Errorf("compute next analysis: %w", err)
	}
	if len(specs) == 0 {
		// All windows are disabled. Fall back to 24 h from now as a safe default
		// rather than returning an error, so the scheduler can still make progress.
		return now.Add(24 * time.Hour), nil
	}

	// --- Step 5: Compute the earliest valid candidate ---
	// The candidate must be:
	//   a) strictly after now
	//   b) >= lastAnalyzedAt + minInterval (when lastAnalyzedAt is non-nil)
	var earliest time.Time
	if lastAnalyzedAt != nil {
		earliest = lastAnalyzedAt.Add(time.Duration(minHours) * time.Hour)
	}

	// Walk day by day from today, skipping weekends for A-share, until we find
	// a window candidate that satisfies both constraints.
	// Safety cap: do not search more than 60 days out.
	const maxDays = 60
	for dayOffset := 0; dayOffset < maxDays; dayOffset++ {
		day := now.AddDate(0, 0, dayOffset)

		// Skip weekends for A-share markets.
		if market == MarketAShare && isWeekend(day) {
			continue
		}

		candidates, err := buildCandidatesForDay(day, specs)
		if err != nil {
			return time.Time{}, fmt.Errorf("compute next analysis: %w", err)
		}

		for _, candidate := range candidates {
			// Constraint (a): must be after now.
			if !candidate.After(now) {
				continue
			}
			// Constraint (b): must satisfy minimum interval.
			if !earliest.IsZero() && candidate.Before(earliest) {
				continue
			}
			return candidate, nil
		}
	}

	// Fallback: return now + minInterval (or now + 24 h as absolute fallback).
	if minHours > 0 {
		return now.Add(time.Duration(minHours) * time.Hour), nil
	}
	return now.Add(24 * time.Hour), nil
}

// resolveFrequency applies the three-level priority to determine the effective
// frequency and its optional custom-days companion.
//
// Priority: holding override > market settings > global settings.
func resolveFrequency(
	override *model.HoldingScheduleOverride,
	settings *model.UserScheduleSettings,
	market string,
) (freq string, freqDays *int32) {
	// Level 1: holding override.
	if override != nil && override.Frequency != nil && *override.Frequency != "" {
		return *override.Frequency, override.FrequencyDays
	}

	// Level 2: market-level frequency from settings.
	switch market {
	case MarketAShare:
		if settings.AShareFrequency != nil && *settings.AShareFrequency != "" {
			return *settings.AShareFrequency, settings.AShareFrequencyDays
		}
	case MarketUSStock:
		if settings.USFrequency != nil && *settings.USFrequency != "" {
			return *settings.USFrequency, settings.USFrequencyDays
		}
	}

	// Level 3: global frequency.
	return settings.GlobalFrequency, settings.GlobalFrequencyDays
}

// resolveWindow returns the effective window override string. An empty string
// means "use the market-level enabled flags from settings" (no override).
func resolveWindow(override *model.HoldingScheduleOverride) string {
	if override == nil || override.Window == nil || *override.Window == "" {
		return ""
	}
	return *override.Window
}
