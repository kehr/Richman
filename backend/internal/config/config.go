package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	App              AppConfig
	Database         DatabaseConfig
	JWT              JWTConfig
	LLM              LLMConfig
	PlatformLLM      PlatformLLMConfig
	CORS             CORSConfig
	Notification     NotificationConfig
	NotificationExt  NotificationExtConfig
	Log              LogConfig
	Datasource       DatasourceConfig
	Analysis         AnalysisConfig
	Richson          RichsonConfig
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Env  string
	Port int
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL string
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

// LLMConfig holds LLM provider settings.
type LLMConfig struct {
	Provider     string
	ClaudeAPIKey string
	ClaudeModel  string
	OpenAIAPIKey string
	OpenAIModel  string
	// Vision capability settings. VisionAPIKey / VisionAPIEndpoint / VisionModel
	// are optional; empty values fall back to provider defaults (e.g. reuse
	// ClaudeAPIKey when the vision provider is "claude").
	VisionProvider    string
	VisionAPIKey      string
	VisionAPIEndpoint string
	VisionModel       string
	VisionTimeout     time.Duration
	// ConfigMasterKey is the 64-character hex master key used by
	// internal/llm.Crypto to AES-256-GCM encrypt per-user API keys before
	// they are written to the llm_configs table. It MUST be exactly 32 raw
	// bytes (64 hex chars). The server boot path calls NewCryptoFromHex and
	// log.Fatals if validation fails, so a misconfigured value crashes fast
	// instead of silently running with plaintext storage.
	ConfigMasterKey string
	// ProbeTimeout bounds every user-provider connectivity probe and every
	// Resolver call to a user provider. Keeping it tight (default 5s) is
	// deliberate: a degraded provider must not stall the synthesis pipeline.
	ProbeTimeout time.Duration
}

// NotificationConfig holds notification channel settings.
type NotificationConfig struct {
	WeChatAppID     string
	WeChatAppSecret string
	FeishuWebhook   string
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPassword    string
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string
	Dir   string
}

// DatasourceConfig holds external market data source settings.
type DatasourceConfig struct {
	AKShareBaseURL string
}

// AnalysisConfig controls analysis task execution behavior.
type AnalysisConfig struct {
	TaskTTLHours          int
	HoldingTimeoutSeconds int
	MaxConcurrentHoldings int
}

// RichsonConfig holds connection settings for the richson Python sidecar.
type RichsonConfig struct {
	BaseURL string
	APIKey  string
}

// PlatformLLMConfig holds the platform-level LLM used for batch analysis and
// weekly insight jobs (separate from per-user LLM configs).
type PlatformLLMConfig struct {
	Provider string // "claude" or "openai"
	APIKey   string
	Model    string
	APIBase  string // optional override for openai_compatible endpoints
}

// CORSConfig holds allowed origin settings for CORS middleware.
type CORSConfig struct {
	AllowedOrigins []string
}

// NotificationExtConfig holds additional notification settings beyond the core
// adapters (SMTP, WeChat, Feishu).
type NotificationExtConfig struct {
	AppBaseURL string // public-facing base URL for unsubscribe links
	SMTPFrom   string // formatted From address (e.g. "Richman <noreply@richman.app>")
}

// Load reads configuration from .env file and environment variables.
// Environment variables take precedence over .env file values.
func Load() (*Config, error) {
	// Load .env file if it exists; ignore error if file is missing.
	_ = godotenv.Load()

	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}

	jwtExpiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}

	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	taskTTLHours, err := strconv.Atoi(getEnv("ANALYSIS_TASK_TTL_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid ANALYSIS_TASK_TTL_HOURS: %w", err)
	}

	holdingTimeoutSeconds, err := strconv.Atoi(getEnv("ANALYSIS_TIMEOUT_SECONDS", "45"))
	if err != nil {
		return nil, fmt.Errorf("invalid ANALYSIS_TIMEOUT_SECONDS: %w", err)
	}

	maxConcurrentHoldings, err := strconv.Atoi(getEnv("ANALYSIS_MAX_CONCURRENT", "4"))
	if err != nil {
		return nil, fmt.Errorf("invalid ANALYSIS_MAX_CONCURRENT: %w", err)
	}

	visionTimeoutSeconds, err := strconv.Atoi(getEnv("LLM_VISION_TIMEOUT_SECONDS", "30"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_VISION_TIMEOUT_SECONDS: %w", err)
	}

	probeTimeout, err := time.ParseDuration(getEnv("LLM_PROBE_TIMEOUT", "5s"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_PROBE_TIMEOUT: %w", err)
	}

	cfg := &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "dev"),
			Port: port,
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
			Expiry: time.Duration(jwtExpiryHours) * time.Hour,
		},
		LLM: LLMConfig{
			Provider:          getEnv("LLM_PROVIDER", "claude"),
			ClaudeAPIKey:      getEnv("CLAUDE_API_KEY", ""),
			ClaudeModel:       getEnv("CLAUDE_MODEL", ""),
			OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
			OpenAIModel:       getEnv("OPENAI_MODEL", ""),
			VisionProvider:    getEnv("LLM_VISION_PROVIDER", "claude"),
			VisionAPIKey:      getEnv("VISION_API_KEY", ""),
			VisionAPIEndpoint: getEnv("VISION_API_ENDPOINT", ""),
			VisionModel:       getEnv("VISION_MODEL", ""),
			VisionTimeout:     time.Duration(visionTimeoutSeconds) * time.Second,
			ConfigMasterKey:   getEnv("LLM_CONFIG_MASTER_KEY", ""),
			ProbeTimeout:      probeTimeout,
		},
		Notification: NotificationConfig{
			WeChatAppID:     getEnv("WECHAT_APP_ID", ""),
			WeChatAppSecret: getEnv("WECHAT_APP_SECRET", ""),
			FeishuWebhook:   getEnv("FEISHU_WEBHOOK_URL", ""),
			SMTPHost:        getEnv("SMTP_HOST", ""),
			SMTPPort:        smtpPort,
			SMTPUser:        getEnv("SMTP_USER", ""),
			SMTPPassword:    getEnv("SMTP_PASSWORD", ""),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			Dir:   getEnv("LOG_DIR", "/var/log/richman"),
		},
		Datasource: DatasourceConfig{
			AKShareBaseURL: getEnv("AKSHARE_BASE_URL", ""),
		},
		Analysis: AnalysisConfig{
			TaskTTLHours:          taskTTLHours,
			HoldingTimeoutSeconds: holdingTimeoutSeconds,
			MaxConcurrentHoldings: maxConcurrentHoldings,
		},
		Richson: RichsonConfig{
			BaseURL: getEnv("RICHSON_BASE_URL", "http://localhost:8100"),
			APIKey:  getEnv("RICHSON_API_KEY", ""),
		},
		PlatformLLM: PlatformLLMConfig{
			Provider: getEnv("PLATFORM_LLM_PROVIDER", "claude"),
			APIKey:   getEnv("PLATFORM_LLM_API_KEY", ""),
			Model:    getEnv("PLATFORM_LLM_MODEL", "claude-sonnet-4-20250514"),
			APIBase:  getEnv("PLATFORM_LLM_API_BASE", ""),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseCommaSeparated(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		},
		NotificationExt: NotificationExtConfig{
			AppBaseURL: getEnv("APP_BASE_URL", ""),
			SMTPFrom:   getEnv("SMTP_FROM", ""),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that required configuration fields are set.
func (c *Config) validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Analysis.TaskTTLHours < 0 {
		return fmt.Errorf("ANALYSIS_TASK_TTL_HOURS cannot be negative")
	}
	if c.Analysis.HoldingTimeoutSeconds < 0 {
		return fmt.Errorf("ANALYSIS_TIMEOUT_SECONDS cannot be negative")
	}
	return nil
}

// IsDev returns true if the application is running in development mode.
// The APP_ENV comparison is case-insensitive.
func (c *Config) IsDev() bool {
	return strings.EqualFold(c.App.Env, "dev")
}

// IsProduction returns true if the application is running in production mode.
// Any APP_ENV other than "dev", "test", or "staging" (case-insensitive) is
// treated as production to fail closed on misconfiguration. This function is
// the single source of truth for dev-only feature gates such as the
// onboarding reset endpoint.
func (c *Config) IsProduction() bool {
	switch strings.ToLower(c.App.Env) {
	case "dev", "test", "staging":
		return false
	default:
		return true
	}
}

// getEnv reads an environment variable or returns the fallback value.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// parseCommaSeparated splits a comma-separated string into a trimmed slice,
// filtering out empty entries.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
