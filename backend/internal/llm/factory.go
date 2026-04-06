package llm

import (
	"fmt"

	"github.com/richman/backend/internal/config"
	"go.uber.org/zap"
)

// ProviderFactory is a function that creates a Provider from config and logger.
type ProviderFactory func(cfg *config.Config, logger *zap.Logger) (Provider, error)

// registry holds the registered provider factories.
var registry = map[string]ProviderFactory{}

// Register adds a provider factory to the registry.
func Register(name string, factory ProviderFactory) {
	registry[name] = factory
}

// NewProvider creates an LLM provider based on the configured provider name.
func NewProvider(cfg *config.Config, logger *zap.Logger) (Provider, error) {
	name := cfg.LLM.Provider
	if name == "" {
		name = "claude"
	}

	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown llm provider: %s", name)
	}

	return factory(cfg, logger)
}
