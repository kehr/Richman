package logger

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/lumberjack.v2"
)

// New creates a new zap.Logger based on the given configuration.
//
// All environments use the same JSON encoder so that log lines are parseable
// by tooling (humanlog, jq, lnav) and dev/prod behavior is identical.
//
// Dev mode differences from prod:
//   - Level: DEBUG (prod: INFO)
//   - Output: stdout only, no file rotation
//   - Global fields: omitted (service/env are obvious in a local dev process)
//
// Recommended dev viewing:
//
//	make dev-pretty          # auto-routes through humanlog if installed
//	make dev | jq .          # ad-hoc pretty-print without extra tools
func New(cfg *config.Config) (*zap.Logger, error) {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)

	var cores []zapcore.Core

	if cfg.IsDev() {
		// Dev: single stdout core at DEBUG level. Global service/env fields are
		// omitted — they add noise when running a single local process.
		cores = append(cores, zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel))
	} else {
		// Prod: stdout at INFO + two rotating file sinks.
		cores = append(cores, zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), zap.InfoLevel))

		appWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Log.Dir + "/app.log",
			MaxSize:    100,
			MaxAge:     30,
			MaxBackups: 10,
			Compress:   true,
		})
		cores = append(cores, zapcore.NewCore(jsonEncoder, appWriter, zap.InfoLevel))

		errorWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Log.Dir + "/error.log",
			MaxSize:    50,
			MaxAge:     90,
			MaxBackups: 20,
			Compress:   true,
		})
		cores = append(cores, zapcore.NewCore(jsonEncoder, errorWriter, zap.ErrorLevel))
	}

	core := zapcore.NewTee(cores...)

	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	}
	// Global fields are useful in prod log aggregation (Loki, Datadog) but
	// unnecessary noise in a local dev process.
	if !cfg.IsDev() {
		opts = append(opts,
			zap.Fields(
				zap.String("service", "richman-api"),
				zap.String("env", cfg.App.Env),
			),
		)
	}

	return zap.New(core, opts...), nil
}

// GetLogger retrieves the request-scoped logger from the Gin context.
// If no logger is found, it returns the global zap logger.
func GetLogger(c *gin.Context) *zap.Logger {
	if l, exists := c.Get("logger"); exists {
		if logger, ok := l.(*zap.Logger); ok {
			return logger
		}
	}
	return zap.L()
}
