package synthesis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/recommendation"
	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

// Synthesizer generates structured decision card content from analysis
// results using the Resolver fallback chain. The Resolver encapsulates the
// three-level user -> system_default -> error walk, so the Synthesizer only
// needs to know about success / failure / malformed-response.
type Synthesizer struct {
	resolver llm.Resolver
	logger   *zap.Logger
}

// NewSynthesizer creates a new Synthesizer. A nil resolver is a valid
// input: Synthesize will short-circuit to the template fallback and record
// meta{Source:"template", ProviderUsed:"none"} so the analysis pipeline
// still produces decision cards when no LLM is configured at all.
func NewSynthesizer(resolver llm.Resolver, logger *zap.Logger) *Synthesizer {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Synthesizer{
		resolver: resolver,
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

// SynthesisMeta records provenance of the generated content so the caller
// can stamp decision_cards.synthesis_source and .provider_used. The three
// Source values encode a ternary contract:
//
//   - "llm"      : the entire output (including the recommendation sub-object)
//     came from the LLM response.
//   - "template" : at least the text fields fell back to the deterministic
//     template; ProviderUsed may still name a layer when the LLM answered
//     but returned malformed JSON.
//   - "mixed"    : text fields are LLM-sourced but the recommendation
//     sub-object fell back to fallbackRecommendation.
//
// ProviderUsed mirrors llm.ProviderLayer as a plain string so callers do not
// need to import llm just to persist the value.
type SynthesisMeta struct {
	Source       string // "llm" | "template" | "mixed"
	ProviderUsed string // "user" | "system_default" | "none"
	LatencyMs    int64
}

// Synthesize generates structured decision card content and reports how it
// was produced via SynthesisMeta. The meta is always non-nil on the happy
// path (nil resolver, resolver error, malformed JSON, etc.) so callers can
// unconditionally dereference it when stamping the persisted card.
func (s *Synthesizer) Synthesize(
	ctx context.Context,
	input *SynthesisInput,
	userID int64,
) (*SynthesisOutput, *SynthesisMeta, error) {
	start := time.Now()

	// Nil resolver short-circuit: no LLM layer is available at all. This is
	// the "dev env without a master key / system default" path; callers must
	// still get a decision card.
	if s.resolver == nil {
		s.logger.Info("llm resolver not configured, using template fallback",
			zap.String("asset", input.AssetCode),
		)
		return templateFallback(input), &SynthesisMeta{
			Source:       "template",
			ProviderUsed: string(llm.LayerNone),
			LatencyMs:    elapsedMs(start),
		}, nil
	}

	prompt := buildSynthesisPrompt(input)

	resolved, err := s.resolver.ResolvedChatCompletion(ctx, userID, llm.ChatRequest{
		SystemPrompt: "You are a financial analysis assistant. " +
			"Generate structured investment analysis summaries. " +
			"Respond only with valid JSON.",
		UserPrompt:  prompt,
		MaxTokens:   2048,
		Temperature: 0.4,
	})
	if err != nil || resolved == nil {
		// Distinguish ErrConsentDenied from real failures only in log level;
		// the downstream card is identical (template + none) either way.
		if errors.Is(err, llm.ErrConsentDenied) {
			s.logger.Info("llm resolver returned consent denied, using template fallback",
				zap.String("asset", input.AssetCode),
				zap.Int64("user_id", userID),
			)
		} else {
			s.logger.Warn("llm resolver failed, using template fallback",
				zap.String("asset", input.AssetCode),
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
		}
		return templateFallback(input), &SynthesisMeta{
			Source:       "template",
			ProviderUsed: string(llm.LayerNone),
			LatencyMs:    elapsedMs(start),
		}, nil
	}

	layer := string(resolved.Layer)

	// Parse the main JSON block. A malformed response degrades to template
	// for the text fields, but we keep the layer that answered so operators
	// can see "the LLM was reachable, the output was just unusable".
	output, parseErr := parseSynthesisResponse(resolved.Response.Content)
	if parseErr != nil {
		s.logger.Warn("failed to parse llm synthesis response, using template fallback",
			zap.String("asset", input.AssetCode),
			zap.String("layer", layer),
			zap.Error(parseErr),
		)
		return templateFallback(input), &SynthesisMeta{
			Source:       "template",
			ProviderUsed: layer,
			LatencyMs:    elapsedMs(start),
		}, nil
	}

	// Try the recommendation sub-object. A missing / malformed sub-object
	// degrades the meta to "mixed": the prose came from the LLM, the
	// recommendation structure came from the deterministic fallback.
	source := "llm"
	if parsed := parseRecommendation(extractJSON(resolved.Response.Content)); parsed != nil {
		ensureRecommendation(parsed, input)
		output.Recommendation = *parsed
	} else {
		s.logger.Warn("llm recommendation sub-object missing or invalid, using fallback",
			zap.String("asset", input.AssetCode),
			zap.String("layer", layer),
		)
		output.Recommendation = fallbackRecommendation(input)
		source = "mixed"
	}

	s.logger.Info("llm synthesis completed",
		zap.String("asset", input.AssetCode),
		zap.String("layer", layer),
		zap.String("source", source),
		zap.Duration("latency", resolved.Response.Latency),
	)

	return output, &SynthesisMeta{
		Source:       source,
		ProviderUsed: layer,
		LatencyMs:    elapsedMs(start),
	}, nil
}

// elapsedMs returns milliseconds since start as an int64. Kept as a helper
// so every meta constructor spells the conversion the same way.
func elapsedMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

func buildSynthesisPrompt(input *SynthesisInput) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Generate a decision card summary for %s (%s, type: %s).\n\n",
		input.AssetName, input.AssetCode, input.AssetType)

	sb.WriteString("Analysis data:\n")
	fmt.Fprintf(&sb, "- Trend: direction=%s, strength=%.2f\n",
		input.Trend.Direction, input.Trend.Strength)
	fmt.Fprintf(&sb, "- Position: assessment=%s, percentile=%.2f\n",
		input.Position.Assessment, input.Position.Percentile)
	fmt.Fprintf(&sb, "- Catalyst: direction=%s, score=%.2f\n",
		input.Catalyst.Direction, input.Catalyst.Score)
	fmt.Fprintf(&sb, "- Weights: trend=%.2f, position=%.2f, catalyst=%.2f\n",
		input.Weights.Trend, input.Weights.Position, input.Weights.Catalyst)
	fmt.Fprintf(&sb, "- Confidence: %.1f%%\n", input.Confidence)
	fmt.Fprintf(&sb, "- Recommendation: %s\n", input.Recommendation)
	fmt.Fprintf(&sb, "- User cost price: %.4f, position ratio: %.2f%%\n",
		input.CostPrice, input.PositionRatio*100)

	if len(input.Catalyst.Events) > 0 {
		sb.WriteString("\nCatalyst events:\n")
		for _, ev := range input.Catalyst.Events {
			fmt.Fprintf(&sb, "  - %s (prob: %.2f, impact: %s)\n",
				ev.Title, ev.Probability, ev.Impact)
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
	switch input.Trend.Direction {
	case analysis.DirectionUpward:
		trendDesc = "upward"
	case analysis.DirectionDownward:
		trendDesc = "downward"
	}

	posDesc := "fairly valued"
	switch input.Position.Assessment {
	case analysis.DirectionBullish:
		posDesc = "undervalued"
	case analysis.DirectionBearish:
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
