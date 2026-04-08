package recommendation

import "testing"

func floatPtr(v float64) *float64 { return &v }

func sampleExecution() Execution {
	sl := 95.0
	tp := 120.0
	return Execution{
		Type:       ExecutionStaged,
		StopLoss:   &sl,
		TakeProfit: &tp,
		ValidDays:  7,
		Steps: []Step{
			{
				Order:        1,
				TriggerType:  TriggerPrice,
				TriggerValue: ">=110",
				DeltaPct:     5.0,
				Rationale:    "first leg",
			},
			{
				Order:        2,
				TriggerType:  TriggerTime,
				TriggerValue: "2026-05-01",
				DeltaPct:     5.0,
				Rationale:    "second leg",
			},
		},
	}
}

func TestFingerprint_Deterministic(t *testing.T) {
	exec := sampleExecution()
	a := Fingerprint(50.0, exec)
	b := Fingerprint(50.0, exec)
	if a != b {
		t.Fatalf("expected stable fingerprint, got %s vs %s", a, b)
	}
	if len(a) != 40 {
		t.Fatalf("expected 40-char SHA-1 hex, got %d chars", len(a))
	}
}

func TestFingerprint_RationaleIgnored(t *testing.T) {
	exec1 := sampleExecution()
	exec2 := sampleExecution()
	exec2.Steps[0].Rationale = "completely different prose"
	exec2.Steps[1].Rationale = "another rewording"

	if Fingerprint(50.0, exec1) != Fingerprint(50.0, exec2) {
		t.Fatal("rationale text must not affect fingerprint")
	}
}

func TestFingerprint_TriggerChangeChangesFingerprint(t *testing.T) {
	base := sampleExecution()
	mutated := sampleExecution()
	mutated.Steps[0].TriggerValue = ">=115"

	if Fingerprint(50.0, base) == Fingerprint(50.0, mutated) {
		t.Fatal("changing TriggerValue must change fingerprint")
	}
}

func TestFingerprint_DeltaChangeChangesFingerprint(t *testing.T) {
	base := sampleExecution()
	mutated := sampleExecution()
	mutated.Steps[1].DeltaPct = 10.0

	if Fingerprint(50.0, base) == Fingerprint(50.0, mutated) {
		t.Fatal("changing DeltaPct must change fingerprint")
	}
}

func TestFingerprint_TargetChangeChangesFingerprint(t *testing.T) {
	exec := sampleExecution()
	if Fingerprint(50.0, exec) == Fingerprint(60.0, exec) {
		t.Fatal("changing targetPositionPct must change fingerprint")
	}
}

func TestFingerprint_NilGuardsHandled(t *testing.T) {
	exec := sampleExecution()
	exec.StopLoss = nil
	exec.TakeProfit = nil
	a := Fingerprint(50.0, exec)
	b := Fingerprint(50.0, exec)
	if a != b {
		t.Fatal("nil stop/take must still produce a stable fingerprint")
	}

	withGuards := sampleExecution()
	if Fingerprint(50.0, exec) == Fingerprint(50.0, withGuards) {
		t.Fatal("nil and present guards must yield different fingerprints")
	}
}

func TestFingerprint_StopLossOnlyDifferent(t *testing.T) {
	a := sampleExecution()
	b := sampleExecution()
	b.StopLoss = floatPtr(90.0)
	if Fingerprint(50.0, a) == Fingerprint(50.0, b) {
		t.Fatal("changing StopLoss must change fingerprint")
	}
}

func TestFingerprint_StepOrderNormalized(t *testing.T) {
	a := sampleExecution()
	b := sampleExecution()
	// swap order of slice entries; Order field unchanged
	b.Steps[0], b.Steps[1] = b.Steps[1], b.Steps[0]
	if Fingerprint(50.0, a) != Fingerprint(50.0, b) {
		t.Fatal("fingerprint must be invariant to slice order when Order is the same")
	}
}

func TestFingerprint_TypeChangeChangesFingerprint(t *testing.T) {
	a := sampleExecution()
	b := sampleExecution()
	b.Type = ExecutionOneShot
	if Fingerprint(50.0, a) == Fingerprint(50.0, b) {
		t.Fatal("changing ExecutionType must change fingerprint")
	}
}
