package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/service/portfolio"
	usersettings "github.com/richman/backend/internal/service/user_settings"
)

// HoldingDTO is the API response shape for a holding row. PositionRatio is
// projected from the database decimal into a float64 percentage so that
// user_settings.AttachAmounts can fill PositionAmount when the user has set
// total_capital_cny (TRD §5.3, PRD §8.3).
type HoldingDTO struct {
	HoldingID      int64     `json:"holdingId"`
	UserID         int64     `json:"userId"`
	AssetCode      string    `json:"assetCode"`
	AssetName      string    `json:"assetName"`
	AssetType      string    `json:"assetType"`
	Category       *string   `json:"category,omitempty"`
	CostPrice      float64   `json:"costPrice"`
	PositionRatio  float64   `json:"positionRatio"`
	PositionAmount *float64  `json:"positionAmount,omitempty"`
	Quantity       float64   `json:"quantity"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// toHoldingDTO projects a model.Holding onto the API response DTO. Decimal
// fields are converted to float64 (the precision required for percentage and
// quantity display in the dashboard is well within float64 range).
func toHoldingDTO(h *model.Holding) HoldingDTO {
	cost, _ := h.CostPrice.Float64()
	pct, _ := h.PositionRatio.Float64()
	qty, _ := h.Quantity.Float64()
	return HoldingDTO{
		HoldingID:     h.HoldingID,
		UserID:        h.UserID,
		AssetCode:     h.AssetCode,
		AssetName:     h.AssetName,
		AssetType:     h.AssetType,
		Category:      h.Category,
		CostPrice:     cost,
		PositionRatio: pct,
		Quantity:      qty,
		CreatedAt:     h.CreatedAt,
		UpdatedAt:     h.UpdatedAt,
	}
}

// TradeDTO is the API response shape for a trade row. Price and Quantity
// are projected from decimal.Decimal into float64 so the JSON payload uses
// plain numbers rather than the quoted string form decimal marshals to. The
// frontend relies on numeric types for summary math and `.toFixed()` calls,
// so shipping the raw decimal would silently break the transactions page.
type TradeDTO struct {
	TradeID   int64     `json:"tradeId"`
	HoldingID int64     `json:"holdingId"`
	UserID    int64     `json:"userId"`
	Direction string    `json:"direction"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	TradedAt  time.Time `json:"tradedAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// toTradeDTO projects a model.Trade onto the API response DTO.
func toTradeDTO(t *model.Trade) TradeDTO {
	price, _ := t.Price.Float64()
	qty, _ := t.Quantity.Float64()
	return TradeDTO{
		TradeID:   t.TradeID,
		HoldingID: t.HoldingID,
		UserID:    t.UserID,
		Direction: t.Direction,
		Price:     price,
		Quantity:  qty,
		TradedAt:  t.TradedAt,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// toTradeDTOs projects a slice of trades.
func toTradeDTOs(trades []model.Trade) []TradeDTO {
	out := make([]TradeDTO, 0, len(trades))
	for i := range trades {
		out = append(out, toTradeDTO(&trades[i]))
	}
	return out
}

// PortfolioHandler handles portfolio-related HTTP requests.
type PortfolioHandler struct {
	portfolioService *portfolio.Service
	capital          CapitalProvider
}

// NewPortfolioHandler creates a new PortfolioHandler. The capital provider
// may be nil; in that case PositionAmount fields are never populated.
func NewPortfolioHandler(portfolioService *portfolio.Service, capital CapitalProvider) *PortfolioHandler {
	return &PortfolioHandler{portfolioService: portfolioService, capital: capital}
}

// resolveCapital fetches the optional total capital for the given user.
// Errors are swallowed so a transient profile read failure does not break
// the holdings list response.
func (h *PortfolioHandler) resolveCapital(c *gin.Context, userID int64) *float64 {
	if h.capital == nil {
		return nil
	}
	totalCap, err := h.capital.GetTotalCapitalCNY(c.Request.Context(), userID)
	if err != nil {
		return nil
	}
	return totalCap
}

// RegisterRoutes registers portfolio routes on the given router group.
// All routes require authentication.
func (h *PortfolioHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	holdings := rg.Group("/holdings", authMiddleware)
	holdings.GET("", h.ListHoldings)
	holdings.POST("", h.CreateHolding)
	holdings.PATCH("/:id", h.UpdateHolding)
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

	capital := h.resolveCapital(c, userID)
	dtos := make([]HoldingDTO, len(holdings))
	for i := range holdings {
		dtos[i] = toHoldingDTO(&holdings[i])
		usersettings.AttachAmounts(&dtos[i], capital)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dtos,
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

	dto := toHoldingDTO(holding)
	usersettings.AttachAmounts(&dto, h.resolveCapital(c, userID))
	c.JSON(http.StatusCreated, gin.H{
		"data": dto,
	})
}

// UpdateHolding handles PATCH /api/v1/holdings/:id.
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

	dto := toHoldingDTO(holding)
	usersettings.AttachAmounts(&dto, h.resolveCapital(c, userID))
	c.JSON(http.StatusOK, gin.H{
		"data": dto,
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
		"data": toTradeDTO(trade),
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
		"data": toTradeDTOs(trades),
	})
}
