package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	inviteSvc "github.com/richman/backend/internal/service/invite"
	"go.uber.org/zap"
)

// InviteHandler handles v2 invite endpoints.
type InviteHandler struct {
	inviteSvc *inviteSvc.Service
	logger    *zap.Logger
}

// NewInviteHandler creates a new InviteHandler.
func NewInviteHandler(inviteSvc *inviteSvc.Service, logger *zap.Logger) *InviteHandler {
	return &InviteHandler{
		inviteSvc: inviteSvc,
		logger:    logger,
	}
}

// getMyCodes handles GET /api/v2/invite/my-codes.
// Returns all invite codes owned by the authenticated user with summary stats.
func (h *InviteHandler) getMyCodes(c *gin.Context) {
	userID := middleware.GetUserID(c)

	resp, err := h.inviteSvc.GetMyCodes(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// getMyInvites handles GET /api/v2/invite/my-invites.
// Returns the list of users invited by the authenticated user.
func (h *InviteHandler) getMyInvites(c *gin.Context) {
	userID := middleware.GetUserID(c)

	resp, err := h.inviteSvc.GetMyInvites(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
