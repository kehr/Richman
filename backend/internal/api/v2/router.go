package v2

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	v2middleware "github.com/richman/backend/internal/api/v2/middleware"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/richson"
	analysisSvc "github.com/richman/backend/internal/service/analysis"
	briefingSvc "github.com/richman/backend/internal/service/briefing"
	feedbackSvc "github.com/richman/backend/internal/service/feedback"
	inviteSvc "github.com/richman/backend/internal/service/invite"
	marketSvc "github.com/richman/backend/internal/service/market"
	userSettingsSvc "github.com/richman/backend/internal/service/user_settings"
	"go.uber.org/zap"
)

const (
	// publicRateLimit is the max requests per minute per IP for public endpoints.
	publicRateLimit = 60
)

// RegisterV2Routes wires all v2 API endpoints onto the provided gin.Engine.
//
// Route ordering within the market group is critical:
// /regime and /overview MUST be registered before /:code so Gin's static
// segments take priority over the wildcard parameter.
func RegisterV2Routes(
	r *gin.Engine,
	richsonClient *richson.Client,
	market *marketSvc.Service,
	briefing *briefingSvc.Service,
	feedback *feedbackSvc.Service,
	holdingAnalyzer *analysisSvc.V2HoldingAnalyzer,
	userSettings *userSettingsSvc.Service,
	invite *inviteSvc.Service,
	analysisJobRepo *repo.AnalysisJobReadRepo,
	_ *repo.AssetAnalysisReadRepo, // reserved: market service provides access internally
	assetRepo *repo.AssetRepo,
	platformLLM *richson.LLMConfig,
	jwtMiddleware gin.HandlerFunc,
	logger *zap.Logger,
) {
	// Build handlers.
	marketH := NewMarketHandler(richsonClient, market, invite, logger)
	eventH := NewEventHandler(richsonClient, logger)
	analysisH := NewAnalysisHandler(richsonClient, holdingAnalyzer, analysisJobRepo, assetRepo, platformLLM, logger)
	briefingH := NewBriefingHandler(briefing, logger)
	feedbackH := NewFeedbackHandler(feedback, logger)
	userH := NewUserHandler(userSettings, logger)
	inviteH := NewInviteHandler(invite, logger)

	// Public rate limiter for all v2 endpoints.
	publicRL := v2middleware.IPRateLimit(publicRateLimit)

	v2 := r.Group("/api/v2")

	// ---- market group (public, rate limited) ----
	marketGroup := v2.Group("/market", publicRL)
	// IMPORTANT: static paths registered before /:code to avoid Gin wildcard conflict.
	marketGroup.GET("/regime", marketH.getMarketRegime)
	marketGroup.GET("/overview", marketH.getMarketOverview)
	// /:code subtree
	marketGroup.GET("/:code", marketH.getAssetDetail)
	marketGroup.GET("/:code/ohlcv", marketH.getAssetOHLCV)
	marketGroup.GET("/:code/scores", marketH.getAssetScores)
	marketGroup.GET("/:code/demo-plan", marketH.getAssetDemoPlan)
	// Share endpoint: JWT optional; try to extract if present, no error if absent.
	marketGroup.GET("/:code/share", optionalAuth(jwtMiddleware), marketH.getAssetShare)

	// ---- events group (public, rate limited) ----
	eventsGroup := v2.Group("/events", publicRL)
	eventsGroup.GET("/radar", eventH.getEventsRadar)

	// ---- analysis group (requires JWT) ----
	analysisGroup := v2.Group("/analysis", jwtMiddleware)
	analysisGroup.POST("/trigger-asset", analysisH.triggerAssetAnalysis)
	analysisGroup.GET("/jobs/:jobId", analysisH.getJobStatus)
	analysisGroup.POST("/holding/:holdingId", analysisH.analyzeHolding)
	// Manual recovery for the daily 06:00 batch when richson was unavailable;
	// admin role required (richman-backend-v2-trd SS8.8).
	analysisGroup.POST("/trigger-batch", middleware.RequireAdmin(), analysisH.triggerBatchAnalysis)

	// ---- briefing (requires JWT) ----
	v2.GET("/briefing", jwtMiddleware, briefingH.getBriefing)

	// ---- feedback (requires JWT) ----
	v2.POST("/feedback", jwtMiddleware, feedbackH.createFeedback)

	// ---- user group (requires JWT) ----
	userGroup := v2.Group("/user", jwtMiddleware)
	userGroup.PATCH("/risk-preference", userH.updateRiskPreference)
	userGroup.GET("/email-push", userH.getEmailPush)
	userGroup.PATCH("/email-push", userH.updateEmailPush)

	// ---- invite group (requires JWT) ----
	inviteGroup := v2.Group("/invite", jwtMiddleware)
	inviteGroup.GET("/my-codes", inviteH.getMyCodes)
	inviteGroup.GET("/my-invites", inviteH.getMyInvites)
}

// optionalAuth returns a middleware that runs the provided auth middleware only
// when an Authorization header is present. When no header is present the
// request continues without setting user context keys; when the header is
// present but invalid, the auth middleware's error response takes effect.
//
// This is used for the share endpoint so unauthenticated visitors can access
// public asset data while authenticated users additionally receive their invite
// code in the response.
func optionalAuth(authMiddleware gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}
		// Header present: validate it; auth middleware handles the error case.
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			// Malformed header: skip auth and continue as unauthenticated.
			c.Next()
			return
		}
		// Run the real auth middleware; it sets userID in context on success.
		authMiddleware(c)
	}
}

// GetUserIDOptional extracts the authenticated user ID from the gin context,
// returning 0 when the context carries no user (unauthenticated request).
// Used by endpoints with optional auth (e.g. share).
func GetUserIDOptional(c *gin.Context) int64 {
	return middleware.GetUserID(c)
}
