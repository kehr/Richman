package diff

import "testing"

// baseSnapshot returns a fully populated snapshot used as the previous
// state in the multi-state priority tests. Helpers below mutate copies of
// the result to construct the "current" side of the diff.
func baseSnapshot() CardSnapshot {
	return CardSnapshot{
		ActionLevel:          0,
		TargetPositionPct:    50.0,
		Confidence:           70.0,
		TrendDirection:       "upward",
		PositionDirection:    "neutral",
		CatalystDirection:    "bullish",
		ExecutionFingerprint: "abc123",
	}
}

func TestCompute_DataDegraded(t *testing.T) {
	prev := baseSnapshot()
	state, delta := Compute(Input{
		Current:            baseSnapshot(),
		Previous:           &prev,
		DataSourceDegraded: true,
	})
	if state != BadgeDataDegraded {
		t.Fatalf("state = %s, want %s", state, BadgeDataDegraded)
	}
	if delta != 0.0 {
		t.Fatalf("delta = %v, want 0.0", delta)
	}
}

func TestCompute_FirstAnalysis(t *testing.T) {
	state, delta := Compute(Input{Current: baseSnapshot(), Previous: nil})
	if state != BadgeFirstAnalysis {
		t.Fatalf("state = %s, want %s", state, BadgeFirstAnalysis)
	}
	if delta != 0.0 {
		t.Fatalf("delta = %v, want 0.0", delta)
	}
}

func TestCompute_ActionUpgrade(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.ActionLevel = 1
	state, delta := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeActionUpgrade {
		t.Fatalf("state = %s, want %s", state, BadgeActionUpgrade)
	}
	if delta != 0.0 {
		t.Fatalf("delta = %v, want 0", delta)
	}
}

func TestCompute_ActionDowngrade(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.ActionLevel = -1
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeActionDowngrade {
		t.Fatalf("state = %s, want %s", state, BadgeActionDowngrade)
	}
}

func TestCompute_SignalFlip_Trend(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.TrendDirection = "downward"
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeSignalFlip {
		t.Fatalf("state = %s, want %s", state, BadgeSignalFlip)
	}
}

func TestCompute_SignalFlip_Position(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.PositionDirection = "bearish"
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeSignalFlip {
		t.Fatalf("state = %s, want %s", state, BadgeSignalFlip)
	}
}

func TestCompute_SignalFlip_Catalyst(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.CatalystDirection = "bearish"
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeSignalFlip {
		t.Fatalf("state = %s, want %s", state, BadgeSignalFlip)
	}
}

func TestCompute_PlanAdjust_Fingerprint(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.ExecutionFingerprint = "deadbeef"
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgePlanAdjust {
		t.Fatalf("state = %s, want %s", state, BadgePlanAdjust)
	}
}

func TestCompute_PlanAdjust_Target(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.TargetPositionPct = 60.0
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgePlanAdjust {
		t.Fatalf("state = %s, want %s", state, BadgePlanAdjust)
	}
}

func TestCompute_ConfidenceShift_Positive(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.Confidence = 85.0 // delta = +15
	state, delta := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeConfidenceShift {
		t.Fatalf("state = %s, want %s", state, BadgeConfidenceShift)
	}
	if delta != 15.0 {
		t.Fatalf("delta = %v, want 15", delta)
	}
}

func TestCompute_ConfidenceShift_Negative(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.Confidence = 55.0 // delta = -15
	state, delta := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeConfidenceShift {
		t.Fatalf("state = %s, want %s", state, BadgeConfidenceShift)
	}
	if delta != -15.0 {
		t.Fatalf("delta = %v, want -15", delta)
	}
}

func TestCompute_ConfidenceShift_BoundaryInclusive(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.Confidence = 80.0 // exactly +10
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeConfidenceShift {
		t.Fatalf("state = %s, want %s (boundary inclusive)", state, BadgeConfidenceShift)
	}
}

func TestCompute_ConfidenceShift_JustBelowBoundary(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.Confidence = 79.9 // delta < 10
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeNone {
		t.Fatalf("state = %s, want %s (sub-threshold)", state, BadgeNone)
	}
}

func TestCompute_None(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.Confidence = 71.0
	state, delta := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeNone {
		t.Fatalf("state = %s, want %s", state, BadgeNone)
	}
	if delta != 1.0 {
		t.Fatalf("delta = %v, want 1", delta)
	}
}

// ---------- multi-state priority tests ----------

// Both data-degraded AND a clearly different action level: rule 1 wins.
func TestPriority_DegradedBeatsActionUpgrade(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.ActionLevel = 2
	cur.Confidence = 99.0
	cur.TrendDirection = "downward"
	cur.ExecutionFingerprint = "different"
	state, _ := Compute(Input{
		Current:            cur,
		Previous:           &prev,
		DataSourceDegraded: true,
	})
	if state != BadgeDataDegraded {
		t.Fatalf("state = %s, want %s (rule 1 priority)", state, BadgeDataDegraded)
	}
}

// First analysis with a confidence value that would otherwise trigger
// confidence-shift: rule 2 wins.
func TestPriority_FirstAnalysisBeatsConfidence(t *testing.T) {
	cur := baseSnapshot()
	cur.Confidence = 95.0
	state, delta := Compute(Input{Current: cur, Previous: nil})
	if state != BadgeFirstAnalysis {
		t.Fatalf("state = %s, want %s (rule 2 priority)", state, BadgeFirstAnalysis)
	}
	if delta != 0.0 {
		t.Fatalf("delta = %v, want 0 on first analysis", delta)
	}
}

// Action upgrade alongside a dimension flip and a plan adjust and a
// confidence swing: rule 3 wins.
func TestPriority_ActionUpgradeBeatsFlipAndPlanAndConfidence(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.ActionLevel = 1
	cur.TrendDirection = "downward"
	cur.ExecutionFingerprint = "different"
	cur.TargetPositionPct = 75.0
	cur.Confidence = 90.0
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeActionUpgrade {
		t.Fatalf("state = %s, want %s (rule 3 priority)", state, BadgeActionUpgrade)
	}
}

// Action level unchanged but a dimension flipped, plan also adjusted,
// and confidence swung: rule 4 wins over rules 5 and 6.
func TestPriority_SignalFlipBeatsPlanAndConfidence(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.CatalystDirection = "bearish"
	cur.ExecutionFingerprint = "different"
	cur.Confidence = 90.0
	state, _ := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgeSignalFlip {
		t.Fatalf("state = %s, want %s (rule 4 priority)", state, BadgeSignalFlip)
	}
}

// Plan adjusted plus a confidence swing, no action / direction changes:
// rule 5 wins over rule 6.
func TestPriority_PlanAdjustBeatsConfidence(t *testing.T) {
	prev := baseSnapshot()
	cur := baseSnapshot()
	cur.ExecutionFingerprint = "different"
	cur.Confidence = 90.0
	state, delta := Compute(Input{Current: cur, Previous: &prev})
	if state != BadgePlanAdjust {
		t.Fatalf("state = %s, want %s (rule 5 priority)", state, BadgePlanAdjust)
	}
	if delta != 20.0 {
		t.Fatalf("delta = %v, want 20", delta)
	}
}
