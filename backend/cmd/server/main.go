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
	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/catalyst"
	"github.com/richman/backend/internal/analysis/confidence"
	"github.com/richman/backend/internal/analysis/position"
	"github.com/richman/backend/internal/analysis/synthesis"
	"github.com/richman/backend/internal/analysis/trend"
	"github.com/richman/backend/internal/analysis/weight"
	"github.com/richman/backend/internal/api/middleware"
	v1 "github.com/richman/backend/internal/api/v1"
	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/logger"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/service/auth"
	"github.com/richman/backend/internal/service/portfolio"
	"go.uber.org/zap"

	// Register LLM provider implementations via init().
	_ "github.com/richman/backend/internal/llm/claude"
	_ "github.com/richman/backend/internal/llm/openai"

	analysisService "github.com/richman/backend/internal/service/analysis"
	decisioncard "github.com/richman/backend/internal/service/decision_card"
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
	assetRepo := repo.NewAssetRepo(dbPool)
	holdingRepo := repo.NewHoldingRepo(dbPool)
	tradeRepo := repo.NewTradeRepo(dbPool)
	cardRepo := repo.NewDecisionCardRepo(dbPool)
	resultRepo := repo.NewAnalysisResultRepo(dbPool)

	// Initialize services
	authService := auth.NewService(userRepo, planRepo, inviteRepo, cfg)
	portfolioService := portfolio.NewService(holdingRepo, tradeRepo)

	// Initialize LLM provider (optional; analysis works in degraded mode without it).
	var llmProvider llm.Provider
	var llmEnhancer *catalyst.LLMEnhancer
	var llmSynthesizer *synthesis.Synthesizer

	llmProvider, err = llm.NewProvider(cfg, zapLogger)
	if err != nil {
		zapLogger.Warn("llm provider not available, analysis will run in degraded mode",
			zap.Error(err),
		)
	}
	if llmProvider != nil {
		llmEnhancer = catalyst.NewLLMEnhancer(llmProvider, zapLogger)
		llmSynthesizer = synthesis.NewSynthesizer(llmProvider, zapLogger)
		zapLogger.Info("llm provider initialized", zap.String("provider", llmProvider.Name()))
	}

	// Fallback synthesizer when LLM is not available.
	if llmSynthesizer == nil {
		// Create a synthesizer that will always use the template fallback.
		// Pass nil provider; Synthesize handles nil gracefully via degraded mode.
		llmSynthesizer = synthesis.NewSynthesizer(nil, zapLogger)
	}

	// Initialize analysis components
	taskStore := analysisService.NewTaskStore()
	analysisSvc := analysisService.NewService(&analysisService.Deps{
		HoldingRepo: holdingRepo,
		CardRepo:    cardRepo,
		ResultRepo:  resultRepo,
		Fetcher:     nil, // Will be set when datasource clients are configured.
		TrendCalc:   trend.NewCalculator(),
		PosCalc:     position.NewCalculator(),
		CatCalc:     catalyst.NewCalculator(),
		LLMEnhancer: llmEnhancer,
		Synthesizer: llmSynthesizer,
		WeightMgr:   weight.NewManager(),
		ConfCalc:    confidence.NewCalculator(),
		Matrix:      analysis.NewMatrix(),
		TaskStore:   taskStore,
		Logger:      zapLogger,
	})

	cardService := decisioncard.NewService(cardRepo)

	// Initialize handlers
	authHandler := v1.NewAuthHandler(authService)
	assetCatalogHandler := v1.NewAssetCatalogHandler(assetRepo)
	portfolioHandler := v1.NewPortfolioHandler(portfolioService)
	analysisHandler := v1.NewAnalysisHandler(analysisSvc)
	taskHandler := v1.NewTaskHandler(taskStore)
	cardHandler := v1.NewDecisionCardHandler(cardService)

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
	assetCatalogHandler.RegisterRoutes(apiV1)
	portfolioHandler.RegisterRoutes(apiV1, authMiddleware)
	analysisHandler.RegisterRoutes(apiV1, authMiddleware)
	taskHandler.RegisterRoutes(apiV1, authMiddleware)
	cardHandler.RegisterRoutes(apiV1, authMiddleware)

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
