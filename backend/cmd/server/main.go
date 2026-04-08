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
	"github.com/richman/backend/internal/datasource"
	akshare "github.com/richman/backend/internal/datasource/akshare"
	polymarket "github.com/richman/backend/internal/datasource/polymarket"
	yahoo "github.com/richman/backend/internal/datasource/yahoo"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/logger"
	"github.com/richman/backend/internal/notification"
	emailAdapter "github.com/richman/backend/internal/notification/adapter/email"
	feishuAdapter "github.com/richman/backend/internal/notification/adapter/feishu"
	wechatAdapter "github.com/richman/backend/internal/notification/adapter/wechat"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/service/auth"
	"github.com/richman/backend/internal/service/portfolio"
	"go.uber.org/zap"

	// Register LLM provider implementations via init().
	_ "github.com/richman/backend/internal/llm/claude"
	_ "github.com/richman/backend/internal/llm/openai"

	analysisService "github.com/richman/backend/internal/service/analysis"
	decisioncard "github.com/richman/backend/internal/service/decision_card"
	notificationSvc "github.com/richman/backend/internal/service/notification"
	onboardingSvc "github.com/richman/backend/internal/service/onboarding"
	screenshotSvc "github.com/richman/backend/internal/service/screenshot"
	usersettingsSvc "github.com/richman/backend/internal/service/user_settings"
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
	cardRepo := repo.NewDecisionCardRepo(dbPool, zapLogger)
	resultRepo := repo.NewAnalysisResultRepo(dbPool)
	taskRepo := repo.NewAnalysisTaskRepo(dbPool)
	notifChannelRepo := repo.NewNotificationChannelRepo(dbPool)
	notifLogRepo := repo.NewNotificationLogRepo(dbPool)

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
		zapLogger.Info("llm provider initialized", zap.String("provider", llmProvider.Name()))
	}
	// Synthesizer is always constructed: when llmProvider is nil, Synthesize
	// short-circuits to the template fallback so the analysis pipeline still
	// produces decision cards in degraded mode.
	llmSynthesizer = synthesis.NewSynthesizer(llmProvider, zapLogger)

	// Initialize vision provider (optional; screenshot recognition
	// degrades to a "failed" response when the provider is unavailable).
	var visionProvider llm.VisionProvider
	visionProvider, err = llm.NewVisionProvider(cfg, zapLogger)
	if err != nil {
		zapLogger.Warn("llm vision provider not available, screenshot recognition will degrade",
			zap.Error(err),
		)
	}
	if visionProvider != nil {
		zapLogger.Info("llm vision provider initialized", zap.String("provider", visionProvider.Name()))
	}
	screenshotService := screenshotSvc.NewService(visionProvider, zapLogger, screenshotSvc.Options{})
	onboardingService := onboardingSvc.NewService(userRepo, cfg)
	userSettingsService := usersettingsSvc.NewService(userRepo)

	// Initialize datasource clients
	akshareClient := akshare.New(cfg.Datasource.AKShareBaseURL, zapLogger)
	yahooClient := yahoo.New(zapLogger)
	polymarketClient := polymarket.New(zapLogger)
	fetcher := datasource.NewFetcher(datasource.FetcherDeps{
		AKShare:    akshareClient,
		Yahoo:      yahooClient,
		Polymarket: polymarketClient,
		Logger:     zapLogger,
	})

	// Initialize analysis components
	taskTTL := time.Duration(cfg.Analysis.TaskTTLHours) * time.Hour
	if taskTTL <= 0 {
		taskTTL = 24 * time.Hour
	}
	taskStore := analysisService.NewTaskStore(taskRepo, taskTTL, zapLogger)
	defer taskStore.Stop()
	analysisTimeout := time.Duration(cfg.Analysis.HoldingTimeoutSeconds) * time.Second
	analysisSvc := analysisService.NewService(&analysisService.Deps{
		HoldingRepo:     holdingRepo,
		CardRepo:        cardRepo,
		ResultRepo:      resultRepo,
		UserRepo:        userRepo,
		Fetcher:         fetcher,
		TrendCalc:       trend.NewCalculator(),
		PosCalc:         position.NewCalculator(),
		CatCalc:         catalyst.NewCalculator(),
		LLMEnhancer:     llmEnhancer,
		Synthesizer:     llmSynthesizer,
		WeightMgr:       weight.NewManager(),
		ConfCalc:        confidence.NewCalculator(),
		Matrix:          analysis.NewMatrix(),
		TaskStore:       taskStore,
		AnalysisTimeout: analysisTimeout,
		MaxConcurrent:   cfg.Analysis.MaxConcurrentHoldings,
		Logger:          zapLogger,
	})

	cardService := decisioncard.NewService(cardRepo)

	// Initialize notification system
	dispatcher := notification.NewDispatcher(zapLogger)
	dispatcher.Register(wechatAdapter.New(
		cfg.Notification.WeChatAppID,
		cfg.Notification.WeChatAppSecret,
		zapLogger,
	))
	dispatcher.Register(feishuAdapter.New(
		cfg.Notification.FeishuWebhook,
		zapLogger,
	))
	dispatcher.Register(emailAdapter.New(
		cfg.Notification.SMTPHost,
		cfg.Notification.SMTPPort,
		cfg.Notification.SMTPUser,
		cfg.Notification.SMTPPassword,
		zapLogger,
	))

	notifService := notificationSvc.NewService(notifChannelRepo, notifLogRepo, dispatcher, zapLogger)

	// Initialize scheduler
	scheduler := analysisService.NewScheduler(analysisSvc, notifService, holdingRepo, userRepo, zapLogger)

	// Initialize handlers
	authHandler := v1.NewAuthHandler(authService)
	assetCatalogHandler := v1.NewAssetCatalogHandler(assetRepo)
	portfolioHandler := v1.NewPortfolioHandler(portfolioService, userSettingsService)
	analysisHandler := v1.NewAnalysisHandler(analysisSvc)
	taskHandler := v1.NewTaskHandler(taskStore)
	cardHandler := v1.NewDecisionCardHandler(cardService, userSettingsService)
	notifHandler := v1.NewNotificationHandler(notifService)
	screenshotHandler := v1.NewScreenshotHandler(screenshotService)
	onboardingHandler := v1.NewOnboardingHandler(onboardingService)
	userSettingsHandler := v1.NewUserSettingsHandler(userSettingsService)

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
	notifHandler.RegisterRoutes(apiV1, authMiddleware)
	screenshotHandler.RegisterRoutes(apiV1, authMiddleware)
	onboardingHandler.RegisterRoutes(apiV1, authMiddleware)
	userSettingsHandler.RegisterRoutes(apiV1, authMiddleware)

	// Start scheduler
	scheduler.Start()

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

	// Stop scheduler
	scheduler.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zapLogger.Fatal("server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("server exited")
}
