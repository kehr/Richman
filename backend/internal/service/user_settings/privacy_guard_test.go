package user_settings

import (
	"strings"
	"testing"
)

type cleanPublicCard struct {
	CardID              string  `json:"cardId"`
	Title               string  `json:"title"`
	PositionRatio       float64 `json:"positionRatio"`
	TargetPositionRatio float64 `json:"targetPositionRatio"`
}

type leakingTotalCapital struct {
	CardID          string  `json:"cardId"`
	TotalCapitalCNY float64 `json:"totalCapitalCny"`
}

type leakingAmount struct {
	CardID         string  `json:"cardId"`
	PositionAmount float64 `json:"positionAmount"`
}

type wrapper struct {
	Card leakingAmount `json:"card"`
}

type sliceWrapper struct {
	Cards []leakingTotalCapital `json:"cards"`
}

type ignoredField struct {
	// json:"-" must be skipped; the field name alone is not inspected.
	HiddenAmount float64 `json:"-"`
	Name         string  `json:"name"`
}

type inferredFromFieldName struct {
	// No json tag → falls back to field name, which contains "Amount".
	PositionAmount float64
	Name           string `json:"name"`
}

func TestAssertNoCapitalLeakage_CleanDTO(t *testing.T) {
	if err := AssertNoCapitalLeakage(&cleanPublicCard{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_NilInput(t *testing.T) {
	if err := AssertNoCapitalLeakage(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_TotalCapitalDetected(t *testing.T) {
	err := AssertNoCapitalLeakage(&leakingTotalCapital{})
	if err == nil || !strings.Contains(err.Error(), "totalcapital") {
		t.Errorf("expected totalcapital leak, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_AmountDetected(t *testing.T) {
	err := AssertNoCapitalLeakage(&leakingAmount{})
	if err == nil || !strings.Contains(err.Error(), "amount") {
		t.Errorf("expected amount leak, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_NestedStruct(t *testing.T) {
	err := AssertNoCapitalLeakage(&wrapper{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "card.positionAmount") {
		t.Errorf("expected nested path, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_EmptySliceStillChecksElementType(t *testing.T) {
	err := AssertNoCapitalLeakage(&sliceWrapper{})
	if err == nil {
		t.Fatal("expected error on element type")
	}
	if !strings.Contains(err.Error(), "totalcapital") {
		t.Errorf("expected totalcapital leak, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_PopulatedSlice(t *testing.T) {
	err := AssertNoCapitalLeakage(&struct {
		Items []leakingAmount `json:"items"`
	}{Items: []leakingAmount{{}}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssertNoCapitalLeakage_IgnoredFieldSkipped(t *testing.T) {
	if err := AssertNoCapitalLeakage(&ignoredField{}); err != nil {
		t.Errorf("json:\"-\" field should not trigger, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_FallbackToFieldName(t *testing.T) {
	err := AssertNoCapitalLeakage(&inferredFromFieldName{})
	if err == nil {
		t.Fatal("expected leak detection via field name")
	}
}

func TestAssertNoCapitalLeakage_NilPointerField(t *testing.T) {
	type holder struct {
		Child *cleanPublicCard `json:"child"`
	}
	if err := AssertNoCapitalLeakage(&holder{}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestAssertNoCapitalLeakage_SliceOfPrimitives(t *testing.T) {
	type holder struct {
		Names []string `json:"names"`
	}
	if err := AssertNoCapitalLeakage(&holder{Names: []string{"a"}}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
