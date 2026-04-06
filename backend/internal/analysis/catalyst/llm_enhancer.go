package catalyst

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

// LLMEnhancer enhances catalyst analysis with LLM-based web search capabilities.
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
// If the LLM call fails, the base result is returned unchanged (degraded mode).
func (e *LLMEnhancer) Enhance(
	ctx context.Context,
	baseResult analysis.CatalystResult,
	assetCode, assetType string,
) (*analysis.CatalystResult, error) {
	prompt := buildCatalystPrompt(assetCode, assetType, baseResult)

	resp, err := e.provider.ChatCompletion(ctx, llm.ChatRequest{
		SystemPrompt: "You are a financial analyst assistant. Respond only with valid JSON.",
		UserPrompt:   prompt,
		MaxTokens:    1024,
		Temperature:  0.3,
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

func buildCatalystPrompt(assetCode, assetType string, base analysis.CatalystResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"Analyze recent events and news that could affect the asset %s (type: %s).\n",
		assetCode, assetType,
	))
	sb.WriteString("The current quantitative catalyst analysis shows:\n")
	sb.WriteString(fmt.Sprintf("- Direction: %s\n", base.Direction))
	sb.WriteString(fmt.Sprintf("- Score: %.2f\n", base.Score))
	if len(base.Events) > 0 {
		sb.WriteString("- Known events:\n")
		for _, ev := range base.Events {
			sb.WriteString(fmt.Sprintf("  - %s (probability: %.2f, impact: %s)\n",
				ev.Title, ev.Probability, ev.Impact))
		}
	}
	sb.WriteString("\nReturn a JSON object with:\n")
	sb.WriteString(`{"events": [{"title": "...", `)
	sb.WriteString(`"impact": "positive|negative|neutral", `)
	sb.WriteString(`"probability": 0.0-1.0}], "score": -1.0 to 1.0}`)
	sb.WriteString("\nInclude only significant recent events not already listed above.")
	return sb.String()
}

func parseCatalystResponse(content string, base analysis.CatalystResult) (*analysis.CatalystResult, error) {
	// Try to extract JSON from the response.
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

	// If LLM provided a score, blend it with the base score.
	if resp.Score != 0 {
		result.Score = (base.Score + resp.Score) / 2
	}

	// Recalculate direction from blended score.
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
