package openai

import (
	"fmt"

	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/llm"
	"go.uber.org/zap"
)

func init() {
	llm.Register("openai", func(cfg *config.Config, logger *zap.Logger) (llm.Provider, error) {
		if cfg.LLM.OpenAIAPIKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY is required when using openai provider")
		}
		opts := []Option{
			WithModel(cfg.LLM.OpenAIModel),
		}
		return NewClient(cfg.LLM.OpenAIAPIKey, logger, opts...), nil
	})
}
