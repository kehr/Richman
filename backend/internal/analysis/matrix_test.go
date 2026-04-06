package analysis

import "testing"

func TestDecideAllBullish(t *testing.T) {
	m := NewMatrix()
	weights := WeightConfig{Trend: 0.33, Position: 0.34, Catalyst: 0.33}

	rec := m.Decide(
		TrendResult{Direction: DirectionUpward, Strength: 1.0},
		PositionResult{Assessment: DirectionBullish, Percentile: 0.1},
		CatalystResult{Direction: DirectionBullish, Score: 0.8},
		weights,
	)

	if rec != RecommendAggressiveAdd {
		t.Errorf("expected aggressive_add for all bullish, got %s", rec)
	}
}

func TestDecideAllBearish(t *testing.T) {
	m := NewMatrix()
	weights := WeightConfig{Trend: 0.33, Position: 0.34, Catalyst: 0.33}

	rec := m.Decide(
		TrendResult{Direction: DirectionDownward, Strength: 1.0},
		PositionResult{Assessment: DirectionBearish, Percentile: 0.9},
		CatalystResult{Direction: DirectionBearish, Score: -0.8},
		weights,
	)

	if rec != RecommendControlPosition {
		t.Errorf("expected control_position for all bearish, got %s", rec)
	}
}

func TestDecideMixed(t *testing.T) {
	m := NewMatrix()
	weights := WeightConfig{Trend: 0.33, Position: 0.34, Catalyst: 0.33}

	rec := m.Decide(
		TrendResult{Direction: DirectionUpward, Strength: 0.5},
		PositionResult{Assessment: DirectionNeutral, Percentile: 0.5},
		CatalystResult{Direction: DirectionBearish, Score: -0.3},
		weights,
	)

	if rec != RecommendHold {
		t.Errorf("expected hold for mixed signals, got %s", rec)
	}
}

func TestDecideMostlyBullish(t *testing.T) {
	m := NewMatrix()
	weights := WeightConfig{Trend: 0.30, Position: 0.40, Catalyst: 0.30}

	rec := m.Decide(
		TrendResult{Direction: DirectionUpward, Strength: 0.8},
		PositionResult{Assessment: DirectionBullish, Percentile: 0.2},
		CatalystResult{Direction: DirectionNeutral, Score: 0.1},
		weights,
	)

	if rec != RecommendSmallAdd && rec != RecommendAggressiveAdd {
		t.Errorf("expected small_add or aggressive_add for mostly bullish, got %s", rec)
	}
}

func TestDecideMostlyBearish(t *testing.T) {
	m := NewMatrix()
	weights := WeightConfig{Trend: 0.30, Position: 0.40, Catalyst: 0.30}

	rec := m.Decide(
		TrendResult{Direction: DirectionDownward, Strength: 0.7},
		PositionResult{Assessment: DirectionBearish, Percentile: 0.85},
		CatalystResult{Direction: DirectionNeutral, Score: -0.1},
		weights,
	)

	if rec != RecommendGradualReduce && rec != RecommendControlPosition {
		t.Errorf("expected gradual_reduce or control_position for mostly bearish, got %s", rec)
	}
}

func TestDecideWeightImpact(t *testing.T) {
	m := NewMatrix()

	// Heavy catalyst weight with bullish catalyst should pull result positive
	heavyCatalyst := WeightConfig{Trend: 0.10, Position: 0.10, Catalyst: 0.80}
	rec := m.Decide(
		TrendResult{Direction: DirectionDownward, Strength: 0.5},
		PositionResult{Assessment: DirectionBearish, Percentile: 0.8},
		CatalystResult{Direction: DirectionBullish, Score: 0.9},
		heavyCatalyst,
	)

	if rec == RecommendControlPosition || rec == RecommendGradualReduce {
		t.Errorf("heavy bullish catalyst weight should prevent bearish recommendation, got %s", rec)
	}
}

func TestDirectionScore(t *testing.T) {
	tests := []struct {
		dir  Direction
		want float64
	}{
		{DirectionUpward, 1.0},
		{DirectionBullish, 1.0},
		{DirectionDownward, -1.0},
		{DirectionBearish, -1.0},
		{DirectionSideways, 0.0},
		{DirectionNeutral, 0.0},
	}

	for _, tt := range tests {
		got := directionScore(tt.dir)
		if got != tt.want {
			t.Errorf("directionScore(%s) = %f, want %f", tt.dir, got, tt.want)
		}
	}
}

func TestScoreToRecommendation(t *testing.T) {
	tests := []struct {
		score float64
		want  Recommendation
	}{
		{0.8, RecommendAggressiveAdd},
		{0.35, RecommendSmallAdd},
		{0.0, RecommendHold},
		{-0.3, RecommendGradualReduce},
		{-0.7, RecommendControlPosition},
	}

	for _, tt := range tests {
		got := scoreToRecommendation(tt.score)
		if got != tt.want {
			t.Errorf("scoreToRecommendation(%f) = %s, want %s", tt.score, got, tt.want)
		}
	}
}
