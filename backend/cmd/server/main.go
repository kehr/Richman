package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	v1 "github.com/richman/backend/internal/api/v1"
	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/logger"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/service/auth"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	zapLogger, err := logger.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() { _ = zapLogger.Sync() }()
	zap.ReplaceGlobals(zapLogger)

	zapLogger.Info("starting server",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.App.Port),
	)

	// Connect to database
	ctx := context.Background()
	dbPool, err := repo.NewDBPool(ctx, cfg)
	if err != nil {
		zapLogger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	zapLogger.Info("database connected")

	// Initialize repos
	userRepo := repo.NewUserRepo(dbPool)
	planRepo := repo.NewPlanRepo(dbPool)
	inviteRepo := repo.NewInviteRepo(dbPool)

	// Initialize services
	authService := auth.NewService(userRepo, planRepo, inviteRepo, cfg)

	// Initialize handlers
	authHandler := v1.NewAuthHandler(authService)

	// Setup Gin
	if !cfg.IsDev() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// Global middleware stack
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS(cfg))
	router.Use(middleware.AccessLog())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	apiV1 := router.Group("/api/v1")
	authMiddleware := middleware.Auth(authService)
	authHandler.RegisterRoutes(apiV1, authMiddleware)

	// Start server
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		zapLogger.Info("server started", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal("server listen failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zapLogger.Fatal("server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("server exited")
}
