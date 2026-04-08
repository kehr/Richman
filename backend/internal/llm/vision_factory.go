package llm

import (
	"fmt"

	"github.com/richman/backend/internal/config"
	"go.uber.org/zap"
)

// VisionProviderFactory is a function that creates a VisionProvider from
// config and logger. Implementations register themselves in init().
type VisionProviderFactory func(cfg *config.Config, logger *zap.Logger) (VisionProvider, error)

// visionRegistry holds the registered vision provider factories.
var visionRegistry = map[string]VisionProviderFactory{}

// RegisterVision adds a vision provider factory to the registry.
func RegisterVision(name string, factory VisionProviderFactory) {
	visionRegistry[name] = factory
}

// NewVisionProvider creates a vision provider based on the configured name.
// Falls back to "claude" if LLM_VISION_PROVIDER is empty.
func NewVisionProvider(cfg *config.Config, logger *zap.Logger) (VisionProvider, error) {
	name := cfg.LLM.VisionProvider
	if name == "" {
		name = "claude"
	}

	factory, ok := visionRegistry[name]
	if !ok {
		return nil, fmt.Errorf("unknown llm vision provider: %s", name)
	}

	return factory(cfg, logger)
}
