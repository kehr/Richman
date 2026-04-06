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
}
