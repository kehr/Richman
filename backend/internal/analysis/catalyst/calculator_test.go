package catalyst

import (
	"testing"
	"time"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/datasource"
)

func defaultMappings() []ImpactMapping {
	return []ImpactMapping{
		{Keywords: []string{"rate cut", "dovish"}, ImpactSign: 1.0, Relevance: 0.9},
		{Keywords: []string{"rate hike", "hawkish"}, ImpactSign: -1.0, Relevance: 0.9},
		{Keywords: []string{"tariff", "trade war"}, ImpactSign: -1.0, Relevance: 0.7},
		{Keywords: []string{"stimulus", "easing"}, ImpactSign: 1.0, Relevance: 0.8},
	}
}

func TestBullishCatalyst(t *testing.T) {
	calc := NewCalculator()
	events := []datasource.EventProbability{
		{
			MarketID:    "1",
			Question:    "Will the Fed announce a rate cut in June?",
			Probability: 0.8,
			Volume:      50000,
			UpdatedAt:   time.Now(),
		},
		{
			MarketID:    "2",
			Question:    "Will new stimulus package be approved?",
			Probability: 0.7,
			Volume:      30000,
			UpdatedAt:   time.Now(),
		},
	}

	result := calc.Calculate(events, defaultMappings())

	if result.Direction != analysis.DirectionBullish {
		t.Errorf("expected bullish direction, got %s", result.Direction)
	}
	if result.Score <= 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 event summaries, got %d", len(result.Events))
	}
}

func TestBearishCatalyst(t *testing.T) {
	calc := NewCalculator()
	events := []datasource.EventProbability{
		{
			MarketID:    "1",
			Question:    "Will the Fed proceed with rate hike?",
			Probability: 0.9,
			Volume:      60000,
			UpdatedAt:   time.Now(),
		},
		{
			MarketID:    "2",
			Question:    "Will new tariff measures be implemented?",
			Probability: 0.75,
			Volume:      40000,
			UpdatedAt:   time.Now(),
		},
	}

	result := calc.Calculate(events, defaultMappings())

	if result.Direction != analysis.DirectionBearish {
		t.Errorf("expected bearish direction, got %s", result.Direction)
	}
	if result.Score >= 0 {
		t.Errorf("expected negative score, got %f", result.Score)
	}
}

func TestNoEvents(t *testing.T) {
	calc := NewCalculator()
	result := calc.Calculate(nil, defaultMappings())

	if result.Direction != analysis.DirectionNeutral {
		t.Errorf("expected neutral for no events, got %s", result.Direction)
	}
	if result.Score != 0 {
		t.Errorf("expected score 0 for no events, got %f", result.Score)
	}
}

func TestUnmatchedEvents(t *testing.T) {
	calc := NewCalculator()
	events := []datasource.EventProbability{
		{
			MarketID:    "1",
			Question:    "Will Mars colony be established by 2030?",
			Probability: 0.1,
			Volume:      5000,
			UpdatedAt:   time.Now(),
		},
	}

	result := calc.Calculate(events, defaultMappings())

	if result.Direction != analysis.DirectionNeutral {
		t.Errorf("expected neutral for unmatched events, got %s", result.Direction)
	}
	if len(result.Events) != 0 {
		t.Errorf("expected 0 matched events, got %d", len(result.Events))
	}
}

func TestContainsKeyword(t *testing.T) {
	tests := []struct {
		text    string
		keyword string
		want    bool
	}{
		{"Will the Fed rate cut happen?", "rate cut", true},
		{"RATE CUT expected", "rate cut", true},
		{"No changes expected", "rate cut", false},
		{"", "rate cut", false},
		{"rate cut", "", true},
	}

	for _, tt := range tests {
		got := containsKeyword(tt.text, tt.keyword)
		if got != tt.want {
			t.Errorf("containsKeyword(%q, %q) = %v, want %v", tt.text, tt.keyword, got, tt.want)
		}
	}
}
