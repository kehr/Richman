package synthesis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/recommendation"
	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

// Synthesizer generates structured decision card content from analysis results using LLM.
type Synthesizer struct {
	provider llm.Provider
	logger   *zap.Logger
}

// NewSynthesizer creates a new Synthesizer.
func NewSynthesizer(provider llm.Provider, logger *zap.Logger) *Synthesizer {
	return &Synthesizer{
		provider: provider,
		logger:   logger,
	}
}

// SynthesisInput contains all data needed to generate decision card content.
type SynthesisInput struct {
	AssetCode      string
	AssetType      string
	AssetName      string
	Trend          analysis.TrendResult
	Position       analysis.PositionResult
	Catalyst       analysis.CatalystResult
	Weights        analysis.WeightConfig
	Confidence     float64
	Recommendation analysis.Recommendation
	CostPrice      float64
	PositionRatio  float64
}

// SynthesisOutput contains the generated content for a decision card.
type SynthesisOutput struct {
	TrendSummary     string                        `json:"trendSummary"`
	PositionSummary  string                        `json:"positionSummary"`
	CatalystSummary  string                        `json:"catalystSummary"`
	ActionAdvice     string                        `json:"actionAdvice"`
	DetailedAdvice   string                        `json:"detailedAdvice"`
	RiskWarnings     []string                      `json:"riskWarnings"`
	TodayHighlights  string                        `json:"todayHighlights"`
	WeightAdjustment string                        `json:"weightAdjustment"`
	Recommendation   recommendation.Recommendation `json:"recommendation"`
}

// Synthesize generates structured decision card content.
// If the LLM fails, a basic template-based output is returned (degraded mode).
func (s *Synthesizer) Synthesize(ctx context.Context, input *SynthesisInput) (*SynthesisOutput, error) {
	prompt := buildSynthesisPrompt(input)

	resp, err := s.provider.ChatCompletion(ctx, llm.ChatRequest{
		SystemPrompt: "You are a financial analysis assistant. " +
			"Generate structured investment analysis summaries. " +
			"Respond only with valid JSON.",
		UserPrompt:  prompt,
		MaxTokens:   2048,
		Temperature: 0.4,
	})
	if err != nil {
		s.logger.Warn("llm synthesis failed, using template fallback",
			zap.String("asset", input.AssetCode),
			zap.Error(err),
		)
		return templateFallback(input), nil
	}

	output, err := parseSynthesisResponse(resp.Content)
	if err != nil {
		s.logger.Warn("failed to parse llm synthesis response, using fallback",
			zap.String("asset", input.AssetCode),
			zap.Error(err),
		)
		return templateFallback(input), nil
	}

	// Try to parse the recommendation sub-object; fall back to template if
	// the LLM omitted or mangled it. The main text fields stay LLM-generated.
	if parsed := parseRecommendation(extractJSON(resp.Content)); parsed != nil {
		ensureRecommendation(parsed, input)
		output.Recommendation = *parsed
	} else {
		s.logger.Warn("llm recommendation sub-object missing or invalid, using fallback",
			zap.String("asset", input.AssetCode),
		)
		output.Recommendation = fallbackRecommendation(input)
	}

	s.logger.Info("llm synthesis completed",
		zap.String("asset", input.AssetCode),
		zap.Duration("latency", resp.Latency),
	)

	return output, nil
}

func buildSynthesisPrompt(input *SynthesisInput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generate a decision card summary for %s (%s, type: %s).\n\n",
		input.AssetName, input.AssetCode, input.AssetType))

	sb.WriteString("Analysis data:\n")
	sb.WriteString(fmt.Sprintf("- Trend: direction=%s, strength=%.2f\n",
		input.Trend.Direction, input.Trend.Strength))
	sb.WriteString(fmt.Sprintf("- Position: assessment=%s, percentile=%.2f\n",
		input.Position.Assessment, input.Position.Percentile))
	sb.WriteString(fmt.Sprintf("- Catalyst: direction=%s, score=%.2f\n",
		input.Catalyst.Direction, input.Catalyst.Score))
	sb.WriteString(fmt.Sprintf("- Weights: trend=%.2f, position=%.2f, catalyst=%.2f\n",
		input.Weights.Trend, input.Weights.Position, input.Weights.Catalyst))
	sb.WriteString(fmt.Sprintf("- Confidence: %.1f%%\n", input.Confidence))
	sb.WriteString(fmt.Sprintf("- Recommendation: %s\n", input.Recommendation))
	sb.WriteString(fmt.Sprintf("- User cost price: %.4f, position ratio: %.2f%%\n",
		input.CostPrice, input.PositionRatio*100))

	if len(input.Catalyst.Events) > 0 {
		sb.WriteString("\nCatalyst events:\n")
		for _, ev := range input.Catalyst.Events {
			sb.WriteString(fmt.Sprintf("  - %s (prob: %.2f, impact: %s)\n",
				ev.Title, ev.Probability, ev.Impact))
		}
	}

	sb.WriteString("\nReturn a JSON object with these fields:\n")
	sb.WriteString(`{
  "trendSummary": "1-2 sentence trend summary",
  "positionSummary": "1-2 sentence position/valuation summary",
  "catalystSummary": "1-2 sentence catalyst summary",
  "actionAdvice": "direction + logic (keep concise)",
  "detailedAdvice": "price range + trigger conditions",
  "riskWarnings": ["risk item 1", "risk item 2"],
  "todayHighlights": "what changed since last analysis",
  "weightAdjustment": "why weights were adjusted (if any)"
}`)
	sb.WriteString(recommendationPromptSection())
	return sb.String()
}

func parseSynthesisResponse(content string) (*SynthesisOutput, error) {
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in llm response")
	}

	var output SynthesisOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return nil, fmt.Errorf("unmarshal synthesis response: %w", err)
	}

	return &output, nil
}

// templateFallback generates a basic output without LLM.
func templateFallback(input *SynthesisInput) *SynthesisOutput {
	trendDesc := "sideways"
	if input.Trend.Direction == analysis.DirectionUpward {
		trendDesc = "upward"
	} else if input.Trend.Direction == analysis.DirectionDownward {
		trendDesc = "downward"
	}

	posDesc := "fairly valued"
	if input.Position.Assessment == analysis.DirectionBullish {
		posDesc = "undervalued"
	} else if input.Position.Assessment == analysis.DirectionBearish {
		posDesc = "overvalued"
	}

	recDesc := recommendationText(input.Recommendation)

	return &SynthesisOutput{
		TrendSummary: fmt.Sprintf("The trend is %s with strength %.0f%%.", trendDesc, input.Trend.Strength*100),
		PositionSummary: fmt.Sprintf(
			"Valuation appears %s (percentile: %.0f%%).",
			posDesc, input.Position.Percentile*100,
		),
		CatalystSummary: input.Catalyst.Summary,
		ActionAdvice:    fmt.Sprintf("Recommendation: %s. Confidence: %.0f%%.", recDesc, input.Confidence),
		DetailedAdvice:  "Detailed analysis unavailable. Please review manually.",
		RiskWarnings:    []string{"Analysis generated without LLM enhancement; confidence may be lower."},
		TodayHighlights: fmt.Sprintf("Analysis generated at %s.", time.Now().Format("2006-01-02 15:04")),
		Recommendation:  fallbackRecommendation(input),
	}
}

func recommendationText(rec analysis.Recommendation) string {
	switch rec {
	case analysis.RecommendAggressiveAdd:
		return "Aggressive add"
	case analysis.RecommendSmallAdd:
		return "Small add"
	case analysis.RecommendHold:
		return "Hold"
	case analysis.RecommendGradualReduce:
		return "Gradual reduce"
	case analysis.RecommendControlPosition:
		return "Control position"
	default:
		return string(rec)
	}
}

// extractJSON attempts to find the first JSON object in a string.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}
