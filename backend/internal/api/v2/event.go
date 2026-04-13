package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/richson"
	"go.uber.org/zap"
)

// EventHandler handles v2 event endpoints.
type EventHandler struct {
	richsonClient *richson.Client
	logger        *zap.Logger
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(richsonClient *richson.Client, logger *zap.Logger) *EventHandler {
	return &EventHandler{
		richsonClient: richsonClient,
		logger:        logger,
	}
}

// getEventsRadar handles GET /api/v2/events/radar.
// Proxies directly to richson GET /events/radar.
func (h *EventHandler) getEventsRadar(c *gin.Context) {
	resp, err := h.richsonClient.GetEventsRadar(c.Request.Context())
	if err != nil {
		re, ok := richson.IsRichsonError(err)
		if ok {
			c.JSON(re.HTTPStatus, gin.H{
				"error": gin.H{"code": re.Code, "message": re.Message},
			})
			return
		}
		h.logger.Error("richson get events radar failed", zap.Error(err))
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
