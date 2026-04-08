package analysis

import (
	"testing"

	"github.com/richman/backend/internal/analysis/diff"
	"github.com/richman/backend/internal/model"
)

// newCard returns a decision card populated with just the fields that
// influence diff.Compute. All other fields are zero.
func newCard(
	actionLevel int,
	targetPct, confidence float64,
	trendDir, posDir, catDir, fingerprint string,
) *model.DecisionCard {
	return &model.DecisionCard{
		ActionLevel:          actionLevel,
		TargetPositionRatio:  targetPct / 100,
		Confidence:           confidence,
		TrendDirection:       trendDir,
		PositionDirection:    posDir,
		CatalystDirection:    catDir,
		ExecutionFingerprint: fingerprint,
	}
}

func TestComputeCardDiff_FirstAnalysis(t *testing.T) {
	cur := newCard(1, 25, 70, "upward", "bullish", "bullish", "fp1")
	badge, delta := computeCardDiff(cur, nil, false)
	if badge != diff.BadgeFirstAnalysis {
		t.Errorf("expected first_analysis, got %q", badge)
	}
	if delta != 0 {
		t.Errorf("expected delta=0, got %v", delta)
	}
}

func TestComputeCardDiff_DataDegraded(t *testing.T) {
	cur := newCard(1, 25, 70, "upward", "bullish", "bullish", "fp1")
	prev := newCard(1, 25, 70, "upward", "bullish", "bullish", "fp1")
	badge, _ := computeCardDiff(cur, prev, true)
	if badge != diff.BadgeDataDegraded {
		t.Errorf("expected data_degraded, got %q", badge)
	}
}

func TestComputeCardDiff_ActionUpgrade(t *testing.T) {
	prev := newCard(0, 20, 60, "upward", "bullish", "bullish", "fp1")
	cur := newCard(1, 25, 62, "upward", "bullish", "bullish", "fp2")
	badge, delta := computeCardDiff(cur, prev, false)
	if badge != diff.BadgeActionUpgrade {
		t.Errorf("expected action_upgrade, got %q", badge)
	}
	if delta != 2 {
		t.Errorf("expected delta=2, got %v", delta)
	}
}

func TestComputeCardDiff_ActionDowngrade(t *testing.T) {
	prev := newCard(1, 25, 60, "upward", "bullish", "bullish", "fp1")
	cur := newCard(-1, 15, 55, "upward", "bullish", "bullish", "fp2")
	badge, _ := computeCardDiff(cur, prev, false)
	if badge != diff.BadgeActionDowngrade {
		t.Errorf("expected action_downgrade, got %q", badge)
	}
}

func TestComputeCardDiff_SignalFlip(t *testing.T) {
	prev := newCard(1, 20, 60, "upward", "bullish", "bullish", "fp1")
	// Same action level and target/fingerprint, but trend direction flipped.
	cur := newCard(1, 20, 60, "downward", "bullish", "bullish", "fp1")
	badge, _ := computeCardDiff(cur, prev, false)
	if badge != diff.BadgeSignalFlip {
		t.Errorf("expected signal_flip, got %q", badge)
	}
}

func TestComputeCardDiff_PlanAdjust(t *testing.T) {
	prev := newCard(1, 20, 60, "upward", "bullish", "bullish", "fp1")
	// Same action level and same directions; only fingerprint moves.
	cur := newCard(1, 20, 60, "upward", "bullish", "bullish", "fp2")
	badge, _ := computeCardDiff(cur, prev, false)
	if badge != diff.BadgePlanAdjust {
		t.Errorf("expected plan_adjust, got %q", badge)
	}
}

func TestComputeCardDiff_ConfidenceShift(t *testing.T) {
	prev := newCard(1, 20, 60, "upward", "bullish", "bullish", "fp1")
	// Everything identical except confidence jumps by 15 (> threshold 10).
	cur := newCard(1, 20, 75, "upward", "bullish", "bullish", "fp1")
	badge, delta := computeCardDiff(cur, prev, false)
	if badge != diff.BadgeConfidenceShift {
		t.Errorf("expected confidence_shift, got %q", badge)
	}
	if delta != 15 {
		t.Errorf("expected delta=15, got %v", delta)
	}
}

func TestComputeCardDiff_None(t *testing.T) {
	prev := newCard(1, 20, 60, "upward", "bullish", "bullish", "fp1")
	cur := newCard(1, 20, 62, "upward", "bullish", "bullish", "fp1")
	badge, _ := computeCardDiff(cur, prev, false)
	if badge != diff.BadgeNone {
		t.Errorf("expected none, got %q", badge)
	}
}

func TestBuildCardSnapshot_DirectFieldCopy(t *testing.T) {
	card := newCard(2, 30, 70, "upward", "bullish", "bearish", "abc")
	snap := buildCardSnapshot(card)
	if snap.ActionLevel != 2 || snap.TargetPositionPct != 30 || snap.Confidence != 70 {
		t.Errorf("numeric fields mismatch: %+v", snap)
	}
	if snap.TrendDirection != "upward" || snap.PositionDirection != "bullish" ||
		snap.CatalystDirection != "bearish" {
		t.Errorf("direction fields mismatch: %+v", snap)
	}
	if snap.ExecutionFingerprint != "abc" {
		t.Errorf("fingerprint mismatch: %q", snap.ExecutionFingerprint)
	}
}
