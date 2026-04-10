package catalyst

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/prompts"
	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

// LLMEnhancer enriches catalyst analysis with event knowledge from the LLM.
type LLMEnhancer struct {
	provider llm.Provider
	logger   *zap.Logger
}

// NewLLMEnhancer creates a new LLM-based catalyst enhancer.
func NewLLMEnhancer(provider llm.Provider, logger *zap.Logger) *LLMEnhancer {
	return &LLMEnhancer{
		provider: provider,
		logger:   logger,
	}
}

// llmCatalystResponse is the expected JSON structure from the LLM.
type llmCatalystResponse struct {
	Events []llmEvent `json:"events"`
	Score  float64    `json:"score"`
}

type llmEvent struct {
	Title       string  `json:"title"`
	Impact      string  `json:"impact"` // positive, negative, neutral
	Probability float64 `json:"probability"`
}

// Enhance enriches a base catalyst result with LLM-derived event insights.
// If the LLM call fails or returns unparseable output, the base result is
// returned unchanged so the analysis pipeline degrades gracefully.
func (e *LLMEnhancer) Enhance(
	ctx context.Context,
	baseResult analysis.CatalystResult,
	assetCode, assetType string,
) (*analysis.CatalystResult, error) {
	userPrompt, err := buildCatalystPrompt(assetCode, assetType, baseResult)
	if err != nil {
		e.logger.Warn("failed to build catalyst prompt, using base result",
			zap.String("asset", assetCode),
			zap.Error(err),
		)
		return &baseResult, nil
	}

	resp, err := e.provider.ChatCompletion(ctx, llm.ChatRequest{
		SystemPrompt: prompts.CatalystSystem(),
		UserPrompt:   userPrompt,
		MaxTokens:    1024,
		Temperature:  0.2,
	})
	if err != nil {
		e.logger.Warn("llm catalyst enhancement failed, using base result",
			zap.String("asset", assetCode),
			zap.Error(err),
		)
		return &baseResult, nil
	}

	enhanced, err := parseCatalystResponse(resp.Content, baseResult)
	if err != nil {
		e.logger.Warn("failed to parse llm catalyst response, using base result",
			zap.String("asset", assetCode),
			zap.Error(err),
		)
		return &baseResult, nil
	}

	e.logger.Info("llm catalyst enhancement applied",
		zap.String("asset", assetCode),
		zap.Int("new_events", len(enhanced.Events)-len(baseResult.Events)),
	)

	return enhanced, nil
}

// buildCatalystPrompt constructs the catalyst user prompt from the template.
// MinScore / MaxScore clamp the score adjustment the LLM is allowed to make,
// preventing a hallucinated extreme score from flipping the quantitative signal.
func buildCatalystPrompt(assetCode, assetType string, base analysis.CatalystResult) (string, error) {
	minScore := base.Score - 0.25
	if minScore < -1.0 {
		minScore = -1.0
	}
	maxScore := base.Score + 0.25
	if maxScore > 1.0 {
		maxScore = 1.0
	}

	events := make([]prompts.CatalystEventData, 0, len(base.Events))
	for _, ev := range base.Events {
		events = append(events, prompts.CatalystEventData{
			Impact:      ev.Impact,
			Probability: ev.Probability,
			Title:       ev.Title,
		})
	}

	return prompts.CatalystUser(&prompts.CatalystData{
		AssetCode: assetCode,
		AssetType: assetType,
		Direction: string(base.Direction),
		Score:     base.Score,
		MinScore:  minScore,
		MaxScore:  maxScore,
		Events:    events,
	})
}

func parseCatalystResponse(content string, base analysis.CatalystResult) (*analysis.CatalystResult, error) {
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in llm response")
	}

	var resp llmCatalystResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal llm response: %w", err)
	}

	// Merge LLM events with base events.
	result := base
	for _, ev := range resp.Events {
		result.Events = append(result.Events, analysis.EventSummary{
			Title:       ev.Title,
			Probability: ev.Probability,
			Impact:      ev.Impact,
		})
	}

	// If LLM provided a non-zero score, blend it with the base score.
	// The LLM score is clamped to ±0.25 of the base before blending so a
	// hallucinated extreme score cannot flip the quantitative signal direction.
	if resp.Score != 0 {
		clampedLLM := resp.Score
		if clampedLLM > base.Score+0.25 {
			clampedLLM = base.Score + 0.25
		}
		if clampedLLM < base.Score-0.25 {
			clampedLLM = base.Score - 0.25
		}
		// Weighted blend: base carries 60% to preserve quantitative signal primacy;
		// LLM event-derived opinion carries 40%.
		result.Score = base.Score*0.6 + clampedLLM*0.4
	}

	// Recalculate direction from the blended score.
	switch {
	case result.Score > 0.2:
		result.Direction = analysis.DirectionBullish
	case result.Score < -0.2:
		result.Direction = analysis.DirectionBearish
	default:
		result.Direction = analysis.DirectionNeutral
	}

	return &result, nil
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
