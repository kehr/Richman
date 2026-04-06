package claude

import (
	"fmt"

	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

func init() {
	llm.Register("claude", func(cfg *config.Config, logger *zap.Logger) (llm.Provider, error) {
		if cfg.LLM.ClaudeAPIKey == "" {
			return nil, fmt.Errorf("CLAUDE_API_KEY is required when using claude provider")
		}
		opts := []Option{
			WithModel(cfg.LLM.ClaudeModel),
		}
		return NewClient(cfg.LLM.ClaudeAPIKey, logger, opts...), nil
	})

	llm.RegisterVision("claude", func(cfg *config.Config, logger *zap.Logger) (llm.VisionProvider, error) {
		// Vision credentials fall back to the text Claude credentials so a
		// single API key can power both capabilities unless explicitly split.
		apiKey := cfg.LLM.VisionAPIKey
		if apiKey == "" {
			apiKey = cfg.LLM.ClaudeAPIKey
		}
		if apiKey == "" {
			return nil, fmt.Errorf("VISION_API_KEY or CLAUDE_API_KEY is required when using claude vision provider")
		}
		opts := []VisionOption{
			WithVisionModel(cfg.LLM.VisionModel),
			WithVisionBaseURL(cfg.LLM.VisionAPIEndpoint),
			WithVisionTimeout(cfg.LLM.VisionTimeout),
		}
		return NewVisionClient(apiKey, logger, opts...), nil
	})
}
