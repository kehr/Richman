package catalyst

import (
	"math"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/datasource"
)

// Calculator computes the catalyst dimension from event probabilities.
// This is the quantitative base only; LLM enhancement is handled separately.
type Calculator struct{}

// NewCalculator creates a new catalyst calculator.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// ImpactMapping defines how to interpret an event's impact on an asset.
type ImpactMapping struct {
	Keywords   []string // keywords that match the event question
	ImpactSign float64  // +1.0 for positive correlation, -1.0 for negative
	Relevance  float64  // 0.0-1.0, how relevant this event type is
}

// Calculate processes event probabilities and returns a CatalystResult.
// Events are scored by their probability and expected impact.
func (c *Calculator) Calculate(
	events []datasource.EventProbability,
	impactMappings []ImpactMapping,
) analysis.CatalystResult {
	if len(events) == 0 {
		return analysis.CatalystResult{
			Direction: analysis.DirectionNeutral,
			Score:     0,
			Summary:   "No catalyst events available for analysis.",
			Events:    nil,
		}
	}

	totalScore := 0.0
	totalWeight := 0.0
	summaries := make([]analysis.EventSummary, 0, len(events))

	for _, event := range events {
		relevance, impactSign := c.matchEvent(event, impactMappings)
		if relevance == 0 {
			continue
		}

		// Score = probability * impact_sign * relevance
		// Higher probability events with higher relevance have more weight
		eventScore := event.Probability * impactSign * relevance
		weight := relevance * event.Volume / (event.Volume + 1000) // volume-weighted confidence

		if weight < 0.01 {
			weight = 0.01 // minimum weight to avoid zero division
		}

		totalScore += eventScore * weight
		totalWeight += weight

		impact := "neutral"
		if impactSign > 0 {
			impact = "positive"
		} else if impactSign < 0 {
			impact = "negative"
		}

		summaries = append(summaries, analysis.EventSummary{
			Title:       event.Question,
			Probability: event.Probability,
			Impact:      impact,
		})
	}

	score := 0.0
	if totalWeight > 0 {
		score = totalScore / totalWeight
	}

	// Clamp to [-1, 1]
	score = math.Max(-1.0, math.Min(1.0, score))

	direction := analysis.DirectionNeutral
	if score > 0.2 {
		direction = analysis.DirectionBullish
	} else if score < -0.2 {
		direction = analysis.DirectionBearish
	}

	summary := buildCatalystSummary(direction, len(summaries))

	return analysis.CatalystResult{
		Direction: direction,
		Score:     score,
		Summary:   summary,
		Events:    summaries,
	}
}

// matchEvent finds the best impact mapping for an event.
// Returns relevance and impact sign.
func (c *Calculator) matchEvent(
	event datasource.EventProbability,
	mappings []ImpactMapping,
) (relevance, impactSign float64) {
	bestRelevance := 0.0
	bestImpactSign := 0.0

	for _, mapping := range mappings {
		for _, kw := range mapping.Keywords {
			if containsKeyword(event.Question, kw) {
				if mapping.Relevance > bestRelevance {
					bestRelevance = mapping.Relevance
					bestImpactSign = mapping.ImpactSign
				}
			}
		}
	}

	return bestRelevance, bestImpactSign
}

// containsKeyword checks if the text contains the keyword (case-insensitive substring match).
func containsKeyword(text, keyword string) bool {
	textLower := toLower(text)
	kwLower := toLower(keyword)

	for i := 0; i <= len(textLower)-len(kwLower); i++ {
		if textLower[i:i+len(kwLower)] == kwLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func buildCatalystSummary(dir analysis.Direction, eventCount int) string {
	switch dir {
	case analysis.DirectionBullish:
		return "Catalyst events indicate a bullish outlook."
	case analysis.DirectionBearish:
		return "Catalyst events indicate a bearish outlook."
	default:
		if eventCount == 0 {
			return "No catalyst events available for analysis."
		}
		return "Catalyst events show mixed or neutral signals."
	}
}
