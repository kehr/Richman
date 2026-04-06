package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/repo"
)

// AssetCatalogHandler handles asset catalog HTTP requests.
type AssetCatalogHandler struct {
	assetRepo *repo.AssetRepo
}

// NewAssetCatalogHandler creates a new AssetCatalogHandler.
func NewAssetCatalogHandler(assetRepo *repo.AssetRepo) *AssetCatalogHandler {
	return &AssetCatalogHandler{assetRepo: assetRepo}
}

// RegisterRoutes registers asset catalog routes on the given router group.
// These are public endpoints; no auth required.
func (h *AssetCatalogHandler) RegisterRoutes(rg *gin.RouterGroup) {
	assets := rg.Group("/assets")
	assets.GET("", h.ListAssets)
	assets.GET("/:code", h.GetAsset)
}

// ListAssets handles GET /api/v1/assets?type=xxx&keyword=xxx.
func (h *AssetCatalogHandler) ListAssets(c *gin.Context) {
	assetType := c.Query("type")
	keyword := c.Query("keyword")

	assets, err := h.assetRepo.ListAssets(c.Request.Context(), assetType, keyword)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": assets,
	})
}

// GetAsset handles GET /api/v1/assets/:code.
func (h *AssetCatalogHandler) GetAsset(c *gin.Context) {
	code := c.Param("code")

	asset, err := h.assetRepo.GetAssetByCode(c.Request.Context(), code)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if asset == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "asset not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": asset,
	})
}
