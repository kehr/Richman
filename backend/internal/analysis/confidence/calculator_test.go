package confidence

import (
	"testing"

	"github.com/richman/backend/internal/analysis"
)

func TestAllAligned(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionUpward},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionBullish},
		Catalyst:       &analysis.CatalystResult{Direction: analysis.DirectionBullish},
		HasLLMCatalyst: true,
	})

	if result < 80 || result > 100 {
		t.Errorf("expected 80-100 for all aligned, got %f", result)
	}
}

func TestTwoAlignedOneConflict(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionUpward},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionBullish},
		Catalyst:       &analysis.CatalystResult{Direction: analysis.DirectionBearish},
		HasLLMCatalyst: true,
	})

	if result < 50 || result > 70 {
		t.Errorf("expected 50-70 for two aligned one conflict, got %f", result)
	}
}

func TestAllDifferent(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionUpward},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionNeutral},
		Catalyst:       &analysis.CatalystResult{Direction: analysis.DirectionBearish},
		HasLLMCatalyst: true,
	})

	if result < 20 || result > 40 {
		t.Errorf("expected 20-40 for all different, got %f", result)
	}
}

func TestMissingDimension(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionUpward},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionBullish},
		Catalyst:       nil,
		HasLLMCatalyst: true,
	})

	// Two aligned (base ~70) minus 20 for missing = ~50
	if result > 60 {
		t.Errorf("expected reduced confidence for missing dimension, got %f", result)
	}
}

func TestNoLLMCatalystPenalty(t *testing.T) {
	calc := NewCalculator()

	withLLM := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionUpward},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionBullish},
		Catalyst:       &analysis.CatalystResult{Direction: analysis.DirectionBullish},
		HasLLMCatalyst: true,
	})

	withoutLLM := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionUpward},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionBullish},
		Catalyst:       &analysis.CatalystResult{Direction: analysis.DirectionBullish},
		HasLLMCatalyst: false,
	})

	if withLLM-withoutLLM != 10 {
		t.Errorf("expected 10 point penalty for no LLM, diff = %f", withLLM-withoutLLM)
	}
}

func TestAllMissing(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(Input{
		Trend:          nil,
		Position:       nil,
		Catalyst:       nil,
		HasLLMCatalyst: false,
	})

	if result != 0 {
		t.Errorf("expected 0 for all missing dimensions, got %f", result)
	}
}

func TestAllNeutral(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionSideways},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionNeutral},
		Catalyst:       &analysis.CatalystResult{Direction: analysis.DirectionNeutral},
		HasLLMCatalyst: true,
	})

	// All neutral is aligned but not strongly, expect moderate confidence
	if result < 50 || result > 70 {
		t.Errorf("expected 50-70 for all neutral, got %f", result)
	}
}

func TestAllBearishAligned(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(Input{
		Trend:          &analysis.TrendResult{Direction: analysis.DirectionDownward},
		Position:       &analysis.PositionResult{Assessment: analysis.DirectionBearish},
		Catalyst:       &analysis.CatalystResult{Direction: analysis.DirectionBearish},
		HasLLMCatalyst: true,
	})

	if result < 80 || result > 100 {
		t.Errorf("expected 80-100 for all bearish aligned, got %f", result)
	}
}
