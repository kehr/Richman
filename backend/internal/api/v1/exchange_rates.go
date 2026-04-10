package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/service/exchangerate"
)

// ExchangeRatesHandler serves GET /api/v1/exchange-rates.
type ExchangeRatesHandler struct {
	svc *exchangerate.Service
}

// NewExchangeRatesHandler constructs the handler.
func NewExchangeRatesHandler(svc *exchangerate.Service) *ExchangeRatesHandler {
	return &ExchangeRatesHandler{svc: svc}
}

// RegisterRoutes registers GET /exchange-rates behind the auth middleware.
func (h *ExchangeRatesHandler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	rg.GET("/exchange-rates", auth, h.Get)
}

// Get returns exchange rates as "1 CNY = X foreign currency".
// On rate fetch failure, rates contains only {"CNY":1.0} so the frontend
// can detect degraded mode without an error response.
func (h *ExchangeRatesHandler) Get(c *gin.Context) {
	rates := h.svc.GetRates(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"rates":     rates.Values,
			"updatedAt": rates.UpdatedAt,
		},
	})
}
