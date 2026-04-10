package schedule

import "time"

// IsEDT reports whether t falls within Eastern Daylight Time (EDT, UTC-4).
// NYSE DST transitions follow the US fixed calendar rules:
//   - EDT starts at 02:00 EST on the second Sunday of March (clocks spring forward)
//   - EST starts at 02:00 EDT on the first Sunday of November (clocks fall back)
//
// t may be in any timezone; the function resolves transitions in terms of
// the UTC-5 (EST) wall clock to determine the boundary.
func IsEDT(t time.Time) bool {
	springForward, fallBack := dstTransitionsForYear(t.Year())

	// Before spring forward or after fall back => EST (winter, not EDT)
	if t.Before(springForward) || !t.Before(fallBack) {
		return false
	}
	return true
}

// NextDSTTransition returns the next moment when EDT/EST status changes
// relative to now. The returned time is always strictly after now.
func NextDSTTransition(now time.Time) time.Time {
	springForward, fallBack := dstTransitionsForYear(now.Year())

	if now.Before(springForward) {
		return springForward
	}
	if now.Before(fallBack) {
		return fallBack
	}

	// Both transitions for this year are in the past; return next year's spring forward.
	nextSpring, _ := dstTransitionsForYear(now.Year() + 1)
	return nextSpring
}

// dstTransitionsForYear returns the spring-forward (EDT start) and fall-back
// (EST start) transition instants for the given year, both expressed as UTC
// moments.
//
// Spring forward: 02:00 EST = 07:00 UTC on the second Sunday of March.
// Fall back:      02:00 EDT = 06:00 UTC on the first Sunday of November.
func dstTransitionsForYear(year int) (springForward, fallBack time.Time) {
	// Second Sunday of March at 02:00 EST (UTC-5) = 07:00 UTC.
	secondSundayMarch := nthWeekdayOfMonth(year, time.March, time.Sunday, 2)
	springForward = time.Date(
		secondSundayMarch.Year(), secondSundayMarch.Month(), secondSundayMarch.Day(),
		7, 0, 0, 0, time.UTC,
	)

	// First Sunday of November at 02:00 EDT (UTC-4) = 06:00 UTC.
	firstSundayNov := nthWeekdayOfMonth(year, time.November, time.Sunday, 1)
	fallBack = time.Date(
		firstSundayNov.Year(), firstSundayNov.Month(), firstSundayNov.Day(),
		6, 0, 0, 0, time.UTC,
	)

	return springForward, fallBack
}

// nthWeekdayOfMonth returns the date of the n-th occurrence (1-based) of the
// given weekday in the given month of the given year. For example,
// nthWeekdayOfMonth(2024, time.March, time.Sunday, 2) returns the second
// Sunday of March 2024.
func nthWeekdayOfMonth(year int, month time.Month, weekday time.Weekday, n int) time.Time {
	// Start at the 1st of the month (UTC).
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	// Advance to the first occurrence of the target weekday.
	offset := int(weekday) - int(first.Weekday())
	if offset < 0 {
		offset += 7
	}
	firstOccurrence := first.AddDate(0, 0, offset)

	// Add (n-1) weeks to reach the n-th occurrence.
	return firstOccurrence.AddDate(0, 0, (n-1)*7)
}
