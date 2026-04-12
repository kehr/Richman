package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/service/quote"
)

// AssetQuoteHandler handles asset quote API requests.
type AssetQuoteHandler struct {
	service *quote.Service
}

// NewAssetQuoteHandler creates a new handler for asset quotes.
func NewAssetQuoteHandler(service *quote.Service) *AssetQuoteHandler {
	return &AssetQuoteHandler{service: service}
}

var validQuoteAssetTypes = map[string]bool{
	"us_stock":         true,
	"gold_etf":         true,
	"a_share_broad":    true,
	"a_share_industry": true,
}

// RegisterRoutes registers the asset quote routes.
func (h *AssetQuoteHandler) RegisterRoutes(g *gin.RouterGroup, authMW gin.HandlerFunc) {
	g.GET("/quotes/:assetType/:assetCode", authMW, h.getQuote)
}

func (h *AssetQuoteHandler) getQuote(c *gin.Context) {
	assetType := c.Param("assetType")
	assetCode := c.Param("assetCode")

	if !validQuoteAssetTypes[assetType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ASSET_TYPE",
				"message": fmt.Sprintf("unsupported asset type: %s", assetType),
			},
		})
		return
	}

	if assetCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "MISSING_ASSET_CODE",
				"message": "asset code is required",
			},
		})
		return
	}

	dto, err := h.service.GetQuote(c.Request.Context(), assetType, assetCode)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": dto})
}
