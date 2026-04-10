package synthesis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/prompts"
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
	Language       string // "en" or "zh"; empty defaults to "en"
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
	// Model and TokensUsed are populated only when Source is "llm" or "mixed"
	// (i.e. when the LLM was reached). They are empty/zero for template-only paths.
	Model      string
	TokensUsed int
	// PromptSnippet and ResponseSnippet carry the first 300 / 500 characters of
	// the user prompt and raw LLM response respectively. They are populated only
	// on the LLM success path and are intended for task-log display; full content
	// is emitted to the structured logger at Debug level.
	PromptSnippet   string
	ResponseSnippet string
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

	userPrompt, err := buildSynthesisPrompt(input)
	if err != nil {
		s.logger.Warn("failed to build synthesis prompt, using template fallback",
			zap.String("asset", input.AssetCode),
			zap.Error(err),
		)
		return templateFallback(input), &SynthesisMeta{
			Source:       "template",
			ProviderUsed: string(llm.LayerNone),
			LatencyMs:    elapsedMs(start),
		}, nil
	}

	langInstruction := "Respond in English."
	if input.Language == "zh" {
		langInstruction = "Respond in Simplified Chinese."
	}

	systemPrompt, err := prompts.SynthesisSystem(prompts.SynthesisSystemData{
		LangInstruction: langInstruction,
	})
	if err != nil {
		s.logger.Warn("failed to render synthesis system prompt, using template fallback",
			zap.String("asset", input.AssetCode),
			zap.Error(err),
		)
		return templateFallback(input), &SynthesisMeta{
			Source:       "template",
			ProviderUsed: string(llm.LayerNone),
			LatencyMs:    elapsedMs(start),
		}, nil
	}

	s.logger.Debug("llm request",
		zap.String("asset", input.AssetCode),
		zap.String("system_prompt", systemPrompt),
		zap.String("user_prompt", userPrompt),
		zap.Int("max_tokens", 2048),
		zap.Float64("temperature", 0.4),
	)

	resolved, err := s.resolver.ResolvedChatCompletion(ctx, userID, llm.ChatRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    2048,
		Temperature:  0.4,
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

	s.logger.Debug("llm response",
		zap.String("asset", input.AssetCode),
		zap.String("layer", layer),
		zap.String("model", resolved.Response.Model),
		zap.Int("tokens_used", resolved.Response.TokensUsed),
		zap.Duration("latency", resolved.Response.Latency),
		zap.String("content", resolved.Response.Content),
	)

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
		Source:          source,
		ProviderUsed:    layer,
		LatencyMs:       elapsedMs(start),
		Model:           resolved.Response.Model,
		TokensUsed:      resolved.Response.TokensUsed,
		PromptSnippet:   snippet(userPrompt, 300),
		ResponseSnippet: snippet(resolved.Response.Content, 500),
	}, nil
}

// elapsedMs returns milliseconds since start as an int64.
func elapsedMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

// buildSynthesisPrompt constructs the synthesis user prompt from templates.
// It populates SynthesisUserData and SynthesisRecommendationData from the
// input, then concatenates the two rendered sections.
func buildSynthesisPrompt(input *SynthesisInput) (string, error) {
	catEvents := make([]prompts.CatalystEventData, 0, len(input.Catalyst.Events))
	for _, ev := range input.Catalyst.Events {
		catEvents = append(catEvents, prompts.CatalystEventData{
			Impact:      ev.Impact,
			Probability: ev.Probability,
			Title:       ev.Title,
		})
	}

	userSection, err := prompts.SynthesisUser(&prompts.SynthesisUserData{
		AssetName:   input.AssetName,
		AssetCode:   input.AssetCode,
		AssetType:   input.AssetType,
		CostPrice:   input.CostPrice,
		PositionPct: input.PositionRatio * 100,

		TrendWeightPct: input.Weights.Trend * 100,
		TrendDirection: string(input.Trend.Direction),
		TrendStrength:  input.Trend.Strength,
		TrendSignals:   prompts.SortedPairs(input.Trend.Signals),
		TrendSummary:   input.Trend.Summary,

		PosWeightPct:  input.Weights.Position * 100,
		PosAssessment: string(input.Position.Assessment),
		PosPercentile: input.Position.Percentile,
		PosMetrics:    prompts.SortedPairs(input.Position.Metrics),
		PosSummary:    input.Position.Summary,

		CatWeightPct: input.Weights.Catalyst * 100,
		CatDirection: string(input.Catalyst.Direction),
		CatScore:     input.Catalyst.Score,
		CatEvents:    catEvents,
		CatSummary:   input.Catalyst.Summary,

		Recommendation: string(input.Recommendation),
		Confidence:     input.Confidence,
	})
	if err != nil {
		return "", err
	}

	recSection, err := buildRecommendationSection(input)
	if err != nil {
		return "", err
	}

	return userSection + recSection, nil
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

// templateFallback generates a minimal structured output without LLM.
// Text fields are intentionally empty — the frontend renders i18n placeholders
// when summaries are absent. Only structured data fields are populated so that
// the recommendation, direction, and weight signals remain usable.
func templateFallback(input *SynthesisInput) *SynthesisOutput {
	return &SynthesisOutput{
		TrendSummary:    "",
		PositionSummary: "",
		CatalystSummary: input.Catalyst.Summary,
		ActionAdvice:    "",
		DetailedAdvice:  "",
		RiskWarnings:    []string{},
		TodayHighlights: "",
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

// snippet returns the first n runes of s, appending "…" when truncated.
// Safe for multi-byte UTF-8 strings.
func snippet(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// extractJSON finds the first complete JSON object in a string.
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
