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
	logger          *zap.Logger
}

// NewAnalysisHandler creates a new AnalysisHandler.
func NewAnalysisHandler(
	richsonClient *richson.Client,
	holdingAnalyzer *analysisSvc.V2HoldingAnalyzer,
	jobRepo *repo.AnalysisJobReadRepo,
	logger *zap.Logger,
) *AnalysisHandler {
	return &AnalysisHandler{
		richsonClient:   richsonClient,
		holdingAnalyzer: holdingAnalyzer,
		jobRepo:         jobRepo,
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
