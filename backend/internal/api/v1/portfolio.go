package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/service/portfolio"
)

// PortfolioHandler handles portfolio-related HTTP requests.
type PortfolioHandler struct {
	portfolioService *portfolio.Service
}

// NewPortfolioHandler creates a new PortfolioHandler.
func NewPortfolioHandler(portfolioService *portfolio.Service) *PortfolioHandler {
	return &PortfolioHandler{portfolioService: portfolioService}
}

// RegisterRoutes registers portfolio routes on the given router group.
// All routes require authentication.
func (h *PortfolioHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	holdings := rg.Group("/holdings", authMiddleware)
	holdings.GET("", h.ListHoldings)
	holdings.POST("", h.CreateHolding)
	holdings.PUT("/:id", h.UpdateHolding)
	holdings.DELETE("/:id", h.DeleteHolding)
	holdings.POST("/:id/trades", h.AddTrade)
	holdings.GET("/:id/trades", h.ListTrades)
}

// ListHoldings handles GET /api/v1/holdings.
func (h *PortfolioHandler) ListHoldings(c *gin.Context) {
	userID := middleware.GetUserID(c)

	holdings, err := h.portfolioService.ListHoldings(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": holdings,
	})
}

// CreateHolding handles POST /api/v1/holdings.
func (h *PortfolioHandler) CreateHolding(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	var input model.CreateHoldingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	holding, err := h.portfolioService.CreateHolding(c.Request.Context(), userID, &input, email)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": holding,
	})
}

// UpdateHolding handles PUT /api/v1/holdings/:id.
func (h *PortfolioHandler) UpdateHolding(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	holdingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid holding id",
			},
		})
		return
	}

	var input model.UpdateHoldingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	holding, err := h.portfolioService.UpdateHolding(c.Request.Context(), userID, holdingID, &input, email)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": holding,
	})
}

// DeleteHolding handles DELETE /api/v1/holdings/:id.
func (h *PortfolioHandler) DeleteHolding(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	holdingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid holding id",
			},
		})
		return
	}

	if err := h.portfolioService.DeleteHolding(c.Request.Context(), userID, holdingID, email); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "holding deleted"},
	})
}

// AddTrade handles POST /api/v1/holdings/:id/trades.
func (h *PortfolioHandler) AddTrade(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email := middleware.GetEmail(c)

	holdingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid holding id",
			},
		})
		return
	}

	var input model.CreateTradeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	trade, err := h.portfolioService.AddTrade(c.Request.Context(), userID, holdingID, &input, email)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": trade,
	})
}

// ListTrades handles GET /api/v1/holdings/:id/trades.
func (h *PortfolioHandler) ListTrades(c *gin.Context) {
	userID := middleware.GetUserID(c)

	holdingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "invalid holding id",
			},
		})
		return
	}

	trades, err := h.portfolioService.ListTrades(c.Request.Context(), userID, holdingID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": trades,
	})
}
