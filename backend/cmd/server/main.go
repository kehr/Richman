package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
	claudeClient "github.com/richman/backend/internal/llm/claude"
	openaiClient "github.com/richman/backend/internal/llm/openai"
	"github.com/richman/backend/internal/logger"
	"github.com/richman/backend/internal/migration"
	"github.com/richman/backend/internal/notification"
	emailAdapter "github.com/richman/backend/internal/notification/adapter/email"
	feishuAdapter "github.com/richman/backend/internal/notification/adapter/feishu"
	wechatAdapter "github.com/richman/backend/internal/notification/adapter/wechat"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/service/auth"
	inviteSvc "github.com/richman/backend/internal/service/invite"
	"github.com/richman/backend/internal/service/portfolio"
	scheduleSvc "github.com/richman/backend/internal/service/schedule"
	"go.uber.org/zap"

	// Note: claude and openai packages register provider factories via
	// their init() functions when first imported. The aliased imports
	// above (claudeClient / openaiClient) pull them in so llm.NewProvider
	// can find them without needing a separate blank import.

	analysisService "github.com/richman/backend/internal/service/analysis"
	decisioncard "github.com/richman/backend/internal/service/decision_card"
	"github.com/richman/backend/internal/service/exchangerate"
	notificationSvc "github.com/richman/backend/internal/service/notification"
	onboardingSvc "github.com/richman/backend/internal/service/onboarding"
	quoteSvc "github.com/richman/backend/internal/service/quote"
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

	// Verify that every migration committed on disk has been applied to the
	// current database. Booting a server with stale schema causes confusing
	// "500 internal server error" responses at runtime whose root cause
	// (e.g. missing column) is hidden several layers deep in the SQL driver.
	// Fail loudly here with a remediation hint instead.
	if err := migration.VerifyCurrent(ctx, dbPool, filepath.Join("db", "migration")); err != nil {
		zapLogger.Fatal("schema drift detected at startup",
			zap.Error(err),
			zap.String("remediation", "run 'make migrate-up' from the backend directory"),
		)
	}
	zapLogger.Info("schema migrations verified up-to-date")

	// Initialize repos
	userRepo := repo.NewUserRepo(dbPool)
	planRepo := repo.NewPlanRepo(dbPool)
	inviteRepo := repo.NewInviteRepo(dbPool)
	userInviteCodeRepo := repo.NewUserInviteCodeRepo(dbPool)
	inviteRewardRepo := repo.NewInviteRewardRepo(dbPool)
	assetRepo := repo.NewAssetRepo(dbPool)
	holdingRepo := repo.NewHoldingRepo(dbPool)
	tradeRepo := repo.NewTradeRepo(dbPool)
	cardRepo := repo.NewDecisionCardRepo(dbPool, zapLogger)
	resultRepo := repo.NewAnalysisResultRepo(dbPool)
	taskRepo := repo.NewAnalysisTaskRepo(dbPool)
	notifChannelRepo := repo.NewNotificationChannelRepo(dbPool)
	notifLogRepo := repo.NewNotificationLogRepo(dbPool)
	llmConfigRepo := repo.NewLLMConfigRepo(dbPool)
	scheduleRepo := repo.NewScheduleRepo(dbPool)

	// Initialize services
	inviteService := inviteSvc.NewService(userInviteCodeRepo, inviteRewardRepo, userRepo, dbPool, zapLogger)
	authService := auth.NewServiceWithInvite(userRepo, planRepo, inviteRepo, inviteService, cfg)
	portfolioService := portfolio.NewService(holdingRepo, tradeRepo)

	// Initialize LLM provider (optional; analysis works in degraded mode
	// without it). systemDefault is the shared fallback LLM Richman
	// operates for users who have not configured their own provider; it
	// is always wired into the Resolver's second layer (gated behind
	// user consent) so dashboards keep showing live LLM cards when the
	// personal layer fails.
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

	// Initialize crypto from the env master key. In dev without a key
	// we warn and continue with crypto=nil; every mutating user-config
	// endpoint then short-circuits with 503 so plaintext keys cannot be
	// silently persisted. Production envs MUST set LLM_CONFIG_MASTER_KEY
	// and should fatal on missing-key to catch misconfiguration early.
	var llmCrypto *llm.Crypto
	if cfg.LLM.ConfigMasterKey == "" {
		if cfg.IsDev() {
			zapLogger.Warn("LLM_CONFIG_MASTER_KEY not set; user-level llm configs disabled")
		} else {
			zapLogger.Fatal("LLM_CONFIG_MASTER_KEY is required in non-dev environments")
		}
	} else {
		llmCrypto, err = llm.NewCryptoFromHex(cfg.LLM.ConfigMasterKey)
		if err != nil {
			zapLogger.Fatal("llm crypto init failed", zap.Error(err))
		}
	}

	// Provider builders: closures over the concrete factory packages so
	// the llm package does not need to import claude / openai (which
	// would create a cycle with the Provider interface). Builders honor
	// the same signature the Resolver and LLMSettingsHandler consume.
	claudeBuilder := func(apiKey, chatModel string) llm.Provider {
		return claudeClient.NewClient(apiKey, zapLogger, claudeClient.WithModel(chatModel))
	}
	openaiBuilder := func(baseURL, apiKey, chatModel string) llm.Provider {
		opts := []openaiClient.Option{openaiClient.WithModel(chatModel)}
		if baseURL != "" {
			opts = append(opts, openaiClient.WithBaseURL(baseURL))
		}
		return openaiClient.NewClient(apiKey, zapLogger, opts...)
	}

	// Resolver: nil when crypto is absent so the Synthesizer's nil-branch
	// fallback takes over. When crypto is present, wire the full three-
	// layer chain so per-user configs and the shared system default can
	// both answer analysis requests.
	var llmResolver llm.Resolver
	if llmCrypto != nil {
		llmResolver = llm.NewResolver(
			llmConfigRepo,
			userRepo,
			llmCrypto,
			llmProvider,
			claudeBuilder,
			openaiBuilder,
			cfg.LLM.ProbeTimeout,
			zapLogger,
		)
	}

	// Synthesizer is always constructed: when the Resolver is nil,
	// Synthesize short-circuits to the template fallback so the analysis
	// pipeline still produces decision cards in degraded mode.
	llmSynthesizer = synthesis.NewSynthesizer(llmResolver, zapLogger)

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
	onboardingService := onboardingSvc.NewService(userRepo)
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
	scheduleService := scheduleSvc.NewService(scheduleRepo, zapLogger)

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
	scheduler := analysisService.NewScheduler(
		analysisSvc, notifService, holdingRepo, cardRepo, userRepo, scheduleService, zapLogger,
	)

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
	llmSettingsHandler := v1.NewLLMSettingsHandler(v1.LLMSettingsDeps{
		ConfigRepo:    llmConfigRepo,
		ConsentRepo:   userRepo,
		Crypto:        llmCrypto,
		ClaudeBuilder: claudeBuilder,
		OpenAIBuilder: openaiBuilder,
		ProbeTimeout:  cfg.LLM.ProbeTimeout,
		Logger:        zapLogger,
	})
	dashboardHandler := v1.NewDashboardHandler(
		llmConfigRepo,
		cardRepo,
		llmProvider != nil,
		zapLogger,
	)
	exchangeRateService := exchangerate.NewService(zapLogger)
	exchangeRatesHandler := v1.NewExchangeRatesHandler(exchangeRateService)
	quoteService := quoteSvc.NewService(
		quoteSvc.NewFetcherAdapter(fetcher),
		zapLogger,
	)
	quoteHandler := v1.NewAssetQuoteHandler(quoteService)
	scheduleHandler := v1.NewScheduleHandler(scheduleService, holdingRepo, cardRepo, scheduler, zapLogger)

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
	llmSettingsHandler.RegisterRoutes(apiV1, authMiddleware)
	dashboardHandler.RegisterRoutes(apiV1, authMiddleware)
	exchangeRatesHandler.RegisterRoutes(apiV1, authMiddleware)
	quoteHandler.RegisterRoutes(apiV1, authMiddleware)
	scheduleHandler.RegisterRoutes(apiV1, authMiddleware)

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
