package v2

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/richson"
	analysisSvc "github.com/richman/backend/internal/service/analysis"
	"go.uber.org/zap"
)

// AnalysisHandler handles v2 analysis endpoints.
type AnalysisHandler struct {
	richsonClient   *richson.Client
	holdingAnalyzer *analysisSvc.V2HoldingAnalyzer
	jobRepo         *repo.AnalysisJobReadRepo
	assetRepo       *repo.AssetRepo
	platformLLM     *richson.LLMConfig
	logger          *zap.Logger
}

// NewAnalysisHandler creates a new AnalysisHandler.
func NewAnalysisHandler(
	richsonClient *richson.Client,
	holdingAnalyzer *analysisSvc.V2HoldingAnalyzer,
	jobRepo *repo.AnalysisJobReadRepo,
	assetRepo *repo.AssetRepo,
	platformLLM *richson.LLMConfig,
	logger *zap.Logger,
) *AnalysisHandler {
	return &AnalysisHandler{
		richsonClient:   richsonClient,
		holdingAnalyzer: holdingAnalyzer,
		jobRepo:         jobRepo,
		assetRepo:       assetRepo,
		platformLLM:     platformLLM,
		logger:          logger,
	}
}

// triggerAssetAnalysisRequest is the request body for POST /api/v2/analysis/trigger-asset.
type triggerAssetAnalysisRequest struct {
	AssetCode string `json:"assetCode" binding:"required"`
	Locale    string `json:"locale"`
}

// triggerAssetAnalysis handles POST /api/v2/analysis/trigger-asset.
// Proxies to richson POST /jobs/analyze-asset, injecting platform LLM config.
func (h *AnalysisHandler) triggerAssetAnalysis(c *gin.Context) {
	var req triggerAssetAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	locale := req.Locale
	if locale == "" {
		locale = "zh"
	}

	richsonReq := richson.TriggerAssetAnalysisRequest{
		AssetCode: req.AssetCode,
		Locale:    locale,
		LLMConfig: nil, // platform default; no per-user LLM config at this endpoint
	}

	resp, err := h.richsonClient.TriggerAssetAnalysis(c.Request.Context(), richsonReq)
	if err != nil {
		re, ok := richson.IsRichsonError(err)
		if ok {
			c.JSON(re.HTTPStatus, gin.H{
				"error": gin.H{"code": re.Code, "message": re.Message},
			})
			return
		}
		h.logger.Error("richson trigger asset analysis failed",
			zap.String("asset_code", req.AssetCode),
			zap.Error(err),
		)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    ErrRichsonUnavailable.Code,
				"message": ErrRichsonUnavailable.Message,
			},
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"data": resp})
}

// getJobStatus handles GET /api/v2/analysis/jobs/:jobId.
// Reads rs_analysis_jobs directly (richson writes, richman reads).
func (h *AnalysisHandler) getJobStatus(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "jobId is required"},
		})
		return
	}

	job, err := h.jobRepo.GetByJobID(c.Request.Context(), jobID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "JOB_NOT_FOUND",
				"message": "analysis job not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": job})
}

// triggerBatchAnalysisRequest is the request body for POST /api/v2/analysis/trigger-batch.
//
// AssetCodes is optional: when empty, all active assets are submitted (mirrors
// the daily 06:00 cron behaviour). Locale defaults to "zh".
type triggerBatchAnalysisRequest struct {
	AssetCodes []string `json:"assetCodes"`
	Locale     string   `json:"locale"`
}

// triggerBatchAnalysis handles POST /api/v2/analysis/trigger-batch.
//
// Manual recovery hook for the daily 06:00 batch when richson was unavailable
// during the cron window (richman-backend-v2-trd SS8.8). Restricted to admin
// role via RequireAdmin middleware.
func (h *AnalysisHandler) triggerBatchAnalysis(c *gin.Context) {
	var req triggerBatchAnalysisRequest
	// Empty body is allowed: default to "all active assets, locale=zh".
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
			})
			return
		}
	}

	locale := req.Locale
	if locale == "" {
		locale = "zh"
	}

	// Resolve the asset list: explicit input overrides the active-set default.
	var batchAssets []richson.BatchAnalyzeAsset
	if len(req.AssetCodes) > 0 {
		batchAssets = make([]richson.BatchAnalyzeAsset, 0, len(req.AssetCodes))
		for _, code := range req.AssetCodes {
			batchAssets = append(batchAssets, richson.BatchAnalyzeAsset{
				AssetCode: code,
				Locale:    locale,
			})
		}
	} else {
		assets, err := h.assetRepo.ListActiveWithType(c.Request.Context(), "")
		if err != nil {
			h.logger.Error("trigger-batch: list active assets failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "failed to load active asset list",
				},
			})
			return
		}
		if len(assets) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{"jobs": []any{}, "skipped": []any{}, "note": "no active assets"},
			})
			return
		}
		batchAssets = make([]richson.BatchAnalyzeAsset, 0, len(assets))
		for _, a := range assets {
			batchAssets = append(batchAssets, richson.BatchAnalyzeAsset{
				AssetCode: a.Code,
				Locale:    locale,
			})
		}
	}

	richsonReq := richson.TriggerBatchAnalysisRequest{
		Assets:    batchAssets,
		LLMConfig: h.platformLLM,
	}

	resp, err := h.richsonClient.TriggerBatchAnalysis(c.Request.Context(), richsonReq)
	if err != nil {
		if re, ok := richson.IsRichsonError(err); ok {
			c.JSON(re.HTTPStatus, gin.H{
				"error": gin.H{"code": re.Code, "message": re.Message},
			})
			return
		}
		h.logger.Error("trigger-batch: richson call failed",
			zap.Int("assets", len(batchAssets)),
			zap.Error(err),
		)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    ErrRichsonUnavailable.Code,
				"message": ErrRichsonUnavailable.Message,
			},
		})
		return
	}

	h.logger.Info("trigger-batch: dispatched",
		zap.Int("assets", len(batchAssets)),
		zap.Int("jobs", len(resp.Jobs)),
		zap.Int("skipped", len(resp.Skipped)),
		zap.Int64("admin_user_id", middleware.GetUserID(c)),
	)
	c.JSON(http.StatusAccepted, gin.H{"data": resp})
}

// analyzeHolding handles POST /api/v2/analysis/holding/:holdingId.
// Runs the full 7-step per-holding analysis via V2HoldingAnalyzer.
func (h *AnalysisHandler) analyzeHolding(c *gin.Context) {
	raw := c.Param("holdingId")
	holdingID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || holdingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid holdingId",
			},
		})
		return
	}

	userID := middleware.GetUserID(c)

	card, svcErr := h.holdingAnalyzer.AnalyzeHolding(c.Request.Context(), userID, holdingID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": card})
}
