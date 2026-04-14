package v2

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/richson"
	inviteSvc "github.com/richman/backend/internal/service/invite"
	marketSvc "github.com/richman/backend/internal/service/market"
	"go.uber.org/zap"
)

// MarketHandler handles v2 market endpoints.
type MarketHandler struct {
	richsonClient *richson.Client
	marketSvc     *marketSvc.Service
	inviteSvc     *inviteSvc.Service
	logger        *zap.Logger
}

// NewMarketHandler creates a new MarketHandler.
func NewMarketHandler(
	richsonClient *richson.Client,
	marketSvc *marketSvc.Service,
	inviteSvc *inviteSvc.Service,
	logger *zap.Logger,
) *MarketHandler {
	return &MarketHandler{
		richsonClient: richsonClient,
		marketSvc:     marketSvc,
		inviteSvc:     inviteSvc,
		logger:        logger,
	}
}

// getMarketRegime handles GET /api/v2/market/regime.
// Proxies directly to richson and returns its response body unchanged.
func (h *MarketHandler) getMarketRegime(c *gin.Context) {
	resp, err := h.richsonClient.GetMarketRegime(c.Request.Context())
	if err != nil {
		re, ok := richson.IsRichsonError(err)
		if ok {
			c.JSON(re.HTTPStatus, gin.H{
				"error": gin.H{"code": re.Code, "message": re.Message},
			})
			return
		}
		h.logger.Error("richson get market regime failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    ErrRichsonUnavailable.Code,
				"message": ErrRichsonUnavailable.Message,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// getMarketOverview handles GET /api/v2/market/overview.
// Aggregation: assembles asset cards grouped by type from richman DB.
func (h *MarketHandler) getMarketOverview(c *gin.Context) {
	overview, err := h.marketSvc.GetOverview(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": overview})
}

// getAssetDetail handles GET /api/v2/market/:code.
// Aggregation: returns asset detail with latest analysis and dimension scores.
func (h *MarketHandler) getAssetDetail(c *gin.Context) {
	code := c.Param("code")
	detail, err := h.marketSvc.GetAssetDetail(c.Request.Context(), code)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

// getAssetOHLCV handles GET /api/v2/market/:code/ohlcv.
// Proxy to richson GET /market/ohlcv/:code, forwarding query params.
func (h *MarketHandler) getAssetOHLCV(c *gin.Context) {
	code := c.Param("code")
	resp, err := h.richsonClient.GetOHLCV(c.Request.Context(), code)
	if err != nil {
		re, ok := richson.IsRichsonError(err)
		if ok {
			c.JSON(re.HTTPStatus, gin.H{
				"error": gin.H{"code": re.Code, "message": re.Message},
			})
			return
		}
		h.logger.Error("richson get ohlcv failed", zap.String("code", code), zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    ErrRichsonUnavailable.Code,
				"message": ErrRichsonUnavailable.Message,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// getAssetScores handles GET /api/v2/market/:code/scores.
// Proxy to richson GET /assets/:code/score-history.
func (h *MarketHandler) getAssetScores(c *gin.Context) {
	code := c.Param("code")
	resp, err := h.richsonClient.GetScoreHistory(c.Request.Context(), code)
	if err != nil {
		re, ok := richson.IsRichsonError(err)
		if ok {
			c.JSON(re.HTTPStatus, gin.H{
				"error": gin.H{"code": re.Code, "message": re.Message},
			})
			return
		}
		h.logger.Error("richson get score history failed", zap.String("code", code), zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    ErrRichsonUnavailable.Code,
				"message": ErrRichsonUnavailable.Message,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// getAssetDemoPlan handles GET /api/v2/market/:code/demo-plan.
// Reads rs_asset_analyses.demo_plan from richman DB; falls back to richson when absent.
func (h *MarketHandler) getAssetDemoPlan(c *gin.Context) {
	code := c.Param("code")
	ctx := c.Request.Context()

	// Try richman DB first: the latest analysis may already have a demo_plan.
	analysis, err := h.marketSvc.GetLatestAnalysisForDemoPlan(ctx, code)
	if err == nil && analysis != nil && len(analysis.DemoPlan) > 0 && string(analysis.DemoPlan) != "null" {
		// Unmarshal the raw JSON so we can embed it in the response envelope.
		var demoPlan interface{}
		if jsonErr := json.Unmarshal(analysis.DemoPlan, &demoPlan); jsonErr == nil {
			c.JSON(http.StatusOK, gin.H{"data": demoPlan})
			return
		}
	}

	// Fallback: request demo plan from richson.
	req := richson.DemoPlanRequest{
		AssetCode: code,
		Language:  "zh", // default locale; callers may pass ?locale= in future
	}
	if lang := c.Query("locale"); lang != "" {
		req.Language = lang
	}

	resp, richsonErr := h.richsonClient.GetDemoPlan(ctx, req)
	if richsonErr != nil {
		re, ok := richson.IsRichsonError(richsonErr)
		if ok {
			c.JSON(re.HTTPStatus, gin.H{
				"error": gin.H{"code": re.Code, "message": re.Message},
			})
			return
		}
		h.logger.Error("richson get demo plan failed", zap.String("code", code), zap.Error(richsonErr))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    ErrRichsonUnavailable.Code,
				"message": ErrRichsonUnavailable.Message,
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// shareData is the response payload for GET /api/v2/market/:code/share.
type shareData struct {
	AssetCode  string  `json:"assetCode"`
	AssetName  string  `json:"assetName,omitempty"`
	InviteCode *string `json:"inviteCode,omitempty"`
}

// getAssetShare handles GET /api/v2/market/:code/share.
// JWT is optional: if present, appends the user's first unused invite code.
func (h *MarketHandler) getAssetShare(c *gin.Context) {
	code := c.Param("code")
	ctx := c.Request.Context()

	detail, err := h.marketSvc.GetAssetDetail(ctx, code)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	payload := shareData{
		AssetCode: detail.Code,
		AssetName: detail.Name,
	}

	// JWT is optional for this endpoint. Try to extract userID from context
	// (set by the optional auth middleware when the Authorization header is present).
	if userIDRaw, exists := c.Get("userID"); exists {
		if userID, ok := userIDRaw.(int64); ok && userID > 0 {
			inviteCode, icErr := h.inviteSvc.GetFirstAvailableCode(ctx, userID)
			if icErr != nil {
				h.logger.Warn("get first available invite code failed",
					zap.Int64("user_id", userID),
					zap.Error(icErr),
				)
			} else if inviteCode != "" {
				payload.InviteCode = &inviteCode
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": payload})
}
