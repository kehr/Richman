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
// In dev mode it uses a console encoder at debug level writing to stdout only.
// In prod mode it uses a JSON encoder at info level, writing to stdout and
// rotated log files (app.log for Info+, error.log for Error+).
func New(cfg *config.Config) (*zap.Logger, error) {
	var cores []zapcore.Core

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	if cfg.IsDev() {
		devEncoderCfg := zap.NewDevelopmentEncoderConfig()
		devEncoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(devEncoderCfg)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel))
	} else {
		jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)

		// Stdout core
		cores = append(cores, zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), zap.InfoLevel))

		// App log file (Info+)
		appWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Log.Dir + "/app.log",
			MaxSize:    100,
			MaxAge:     30,
			MaxBackups: 10,
			Compress:   true,
		})
		cores = append(cores, zapcore.NewCore(jsonEncoder, appWriter, zap.InfoLevel))

		// Error log file (Error+)
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
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
		zap.Fields(
			zap.String("service", "richman-api"),
			zap.String("env", cfg.App.Env),
		),
	)

	return logger, nil
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
