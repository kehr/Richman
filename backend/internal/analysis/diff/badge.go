// Package diff implements the badge state machine that compares the
// previous and current snapshots of an asset card and emits exactly one
// BadgeState plus the confidence delta. The algorithm follows the strict
// priority order defined in PRD §3.4 and TRD §3.2.
package diff

import (
	"math"

	"github.com/richman/backend/internal/analysis/recommendation"
)

// BadgeState is the discrete badge displayed on a card after a diff.
// At most one state is emitted per Compute call; ties are broken by the
// priority order documented in Compute.
type BadgeState string

const (
	BadgeDataDegraded    BadgeState = "data_degraded"
	BadgeFirstAnalysis   BadgeState = "first_analysis"
	BadgeActionUpgrade   BadgeState = "action_upgrade"
	BadgeActionDowngrade BadgeState = "action_downgrade"
	BadgeSignalFlip      BadgeState = "signal_flip"
	BadgePlanAdjust      BadgeState = "plan_adjust"
	BadgeConfidenceShift BadgeState = "confidence_shift"
	BadgeNone            BadgeState = "none"
)

// CardSnapshot is the minimal slice of card state required by the diff
// algorithm. It is intentionally decoupled from the persistence model so
// the diff package never imports db / repo layers.
type CardSnapshot struct {
	ActionLevel          int
	TargetPositionPct    float64
	Confidence           float64
	TrendDirection       string
	PositionDirection    string
	CatalystDirection    string
	ExecutionFingerprint string
}

// Input bundles everything Compute needs: the current snapshot, the
// optional previous snapshot (nil for the first analysis of an asset),
// and the data-source health flag bubbled up from the ingestion layer.
type Input struct {
	Current            CardSnapshot
	Previous           *CardSnapshot
	DataSourceDegraded bool
}

// Compute returns the badge state and the confidence delta
// (current - previous) for a card transition.
//
// Priority order (first match wins):
//
//  1. DataSourceDegraded         -> BadgeDataDegraded
//  2. Previous == nil            -> BadgeFirstAnalysis
//  3. ActionLevel changed        -> BadgeActionUpgrade / BadgeActionDowngrade
//  4. Any dimension direction changed (and ActionLevel did not)
//     -> BadgeSignalFlip
//  5. ExecutionFingerprint or TargetPositionPct changed
//     -> BadgePlanAdjust
//  6. |confidenceDelta| >= ConfidenceShiftThreshold
//     -> BadgeConfidenceShift
//  7. Otherwise                   -> BadgeNone
//
// On the first analysis the confidence delta is defined as 0.0 because
// there is no baseline to compare against.
func Compute(in Input) (BadgeState, float64) {
	// Rule 1: data quality trumps every other transition.
	if in.DataSourceDegraded {
		return BadgeDataDegraded, 0.0
	}

	// Rule 2: no prior snapshot means a first analysis.
	if in.Previous == nil {
		return BadgeFirstAnalysis, 0.0
	}

	prev := *in.Previous
	cur := in.Current
	delta := cur.Confidence - prev.Confidence

	// Rule 3: action-level shift.
	if cur.ActionLevel > prev.ActionLevel {
		return BadgeActionUpgrade, delta
	}
	if cur.ActionLevel < prev.ActionLevel {
		return BadgeActionDowngrade, delta
	}

	// Rule 4: dimension direction flip while action level holds steady.
	if cur.TrendDirection != prev.TrendDirection ||
		cur.PositionDirection != prev.PositionDirection ||
		cur.CatalystDirection != prev.CatalystDirection {
		return BadgeSignalFlip, delta
	}

	// Rule 5: execution plan or target position adjusted.
	if cur.ExecutionFingerprint != prev.ExecutionFingerprint ||
		cur.TargetPositionPct != prev.TargetPositionPct {
		return BadgePlanAdjust, delta
	}

	// Rule 6: confidence swing exceeds the configured threshold.
	if math.Abs(delta) >= recommendation.ConfidenceShiftThreshold {
		return BadgeConfidenceShift, delta
	}

	// Rule 7: nothing material changed.
	return BadgeNone, delta
}
