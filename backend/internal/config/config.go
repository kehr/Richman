package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	App          AppConfig
	Database     DatabaseConfig
	JWT          JWTConfig
	LLM          LLMConfig
	Notification NotificationConfig
	Log          LogConfig
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
	OpenAIAPIKey string
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
			Provider:     getEnv("LLM_PROVIDER", "claude"),
			ClaudeAPIKey: getEnv("CLAUDE_API_KEY", ""),
			OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
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
	return nil
}

// IsDev returns true if the application is running in development mode.
func (c *Config) IsDev() bool {
	return c.App.Env == "dev"
}

// getEnv reads an environment variable or returns the fallback value.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
