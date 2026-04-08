package user_settings

import "testing"

type dashboardDTO struct {
	PositionRatio        float64  `json:"positionRatio"`
	PositionAmount       *float64 `json:"positionAmount,omitempty"`
	TargetPositionRatio  float64  `json:"targetPositionRatio"`
	TargetPositionAmount *float64 `json:"targetPositionAmount,omitempty"`
	UnrealizedPct        float64  `json:"unrealizedPct"`
	UnrealizedAmount     *float64 `json:"unrealizedAmount,omitempty"`
}

type percentOnly struct {
	SomePct float64 `json:"somePct"`
	// No matching Amount field on purpose.
}

type noPctFields struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type wrongAmountType struct {
	FooPct    float64 `json:"fooPct"`
	FooAmount float64 `json:"fooAmount"` // not *float64, should be skipped
}

func TestAttachAmounts_NilCapitalIsNoop(t *testing.T) {
	dto := &dashboardDTO{PositionRatio: 30}
	AttachAmounts(dto, nil)
	if dto.PositionAmount != nil {
		t.Errorf("expected nil PositionAmount, got %v", *dto.PositionAmount)
	}
}

func TestAttachAmounts_FillsAllPctFields(t *testing.T) {
	cap := 100000.0
	dto := &dashboardDTO{
		PositionRatio:       30,
		TargetPositionRatio: 45,
		UnrealizedPct:       10,
	}
	AttachAmounts(dto, &cap)
	if dto.PositionAmount == nil || *dto.PositionAmount != 30000 {
		t.Errorf("PositionAmount: want 30000, got %v", dto.PositionAmount)
	}
	if dto.TargetPositionAmount == nil || *dto.TargetPositionAmount != 45000 {
		t.Errorf("TargetPositionAmount: want 45000, got %v", dto.TargetPositionAmount)
	}
	if dto.UnrealizedAmount == nil || *dto.UnrealizedAmount != 10000 {
		t.Errorf("UnrealizedAmount: want 10000, got %v", dto.UnrealizedAmount)
	}
}

func TestAttachAmounts_NoPctFieldsUnchanged(t *testing.T) {
	cap := 100000.0
	dto := &noPctFields{Name: "hi", Count: 3}
	AttachAmounts(dto, &cap)
	if dto.Name != "hi" || dto.Count != 3 {
		t.Errorf("unrelated fields mutated: %+v", dto)
	}
}

func TestAttachAmounts_NoMatchingAmountFieldSkipped(t *testing.T) {
	cap := 100000.0
	dto := &percentOnly{SomePct: 50}
	// Should not panic even though SomeAmount does not exist.
	AttachAmounts(dto, &cap)
	if dto.SomePct != 50 {
		t.Errorf("SomePct mutated: %v", dto.SomePct)
	}
}

func TestAttachAmounts_WrongAmountTypeSkipped(t *testing.T) {
	cap := 100000.0
	dto := &wrongAmountType{FooPct: 25}
	AttachAmounts(dto, &cap)
	if dto.FooAmount != 0 {
		t.Errorf("FooAmount (non-pointer) should be skipped, got %v", dto.FooAmount)
	}
}

func TestAttachAmounts_NilDTOIsNoop(t *testing.T) {
	cap := 100000.0
	AttachAmounts(nil, &cap)
	var p *dashboardDTO
	AttachAmounts(p, &cap) // typed nil pointer
}

func TestAttachAmounts_NonPointerIsNoop(t *testing.T) {
	cap := 100000.0
	dto := dashboardDTO{PositionRatio: 30}
	AttachAmounts(dto, &cap) // value, not pointer
	if dto.PositionAmount != nil {
		t.Errorf("value receiver should not be mutated")
	}
}

func TestAttachAmounts_PointerToNonStructIsNoop(t *testing.T) {
	cap := 100000.0
	x := 1
	AttachAmounts(&x, &cap)
}

type pointerPct struct {
	PositionPct    *float64 `json:"positionPct,omitempty"`
	PositionAmount *float64 `json:"positionAmount,omitempty"`
}

func TestAttachAmounts_PointerPctField(t *testing.T) {
	cap := 100000.0
	pct := 40.0
	dto := &pointerPct{PositionPct: &pct}
	AttachAmounts(dto, &cap)
	if dto.PositionAmount == nil || *dto.PositionAmount != 40000 {
		t.Errorf("want 40000, got %v", dto.PositionAmount)
	}
}

func TestAttachAmounts_NilPointerPctSkipped(t *testing.T) {
	cap := 100000.0
	dto := &pointerPct{}
	AttachAmounts(dto, &cap)
	if dto.PositionAmount != nil {
		t.Errorf("nil Pct pointer should not produce an Amount")
	}
}

type float32Pct struct {
	FooPct    float32  `json:"fooPct"`
	FooAmount *float64 `json:"fooAmount,omitempty"`
}

func TestAttachAmounts_Float32PctField(t *testing.T) {
	cap := 100000.0
	dto := &float32Pct{FooPct: 25}
	AttachAmounts(dto, &cap)
	if dto.FooAmount == nil || *dto.FooAmount != 25000 {
		t.Errorf("want 25000, got %v", dto.FooAmount)
	}
}

type stringPct struct {
	// Non-numeric field with Pct suffix — should be silently skipped.
	WeirdPct    string   `json:"weirdPct"`
	WeirdAmount *float64 `json:"weirdAmount,omitempty"`
}

func TestAttachAmounts_NonNumericPctSkipped(t *testing.T) {
	cap := 100000.0
	dto := &stringPct{WeirdPct: "hi"}
	AttachAmounts(dto, &cap)
	if dto.WeirdAmount != nil {
		t.Errorf("non-numeric Pct should be skipped")
	}
}

func TestAmountFieldFor(t *testing.T) {
	cases := map[string]string{
		"PositionPct":         "PositionAmount",
		"UnrealizedPct":       "UnrealizedAmount",
		"TargetPositionPct":   "TargetPositionAmount",
		"PositionRatio":       "PositionAmount",
		"TargetPositionRatio": "TargetPositionAmount",
		"SomeOtherField":      "",
		"Pct":                 "Amount",
	}
	for in, want := range cases {
		if got := amountFieldFor(in); got != want {
			t.Errorf("amountFieldFor(%q) = %q, want %q", in, got, want)
		}
	}
}
