package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/service/auth"
	"go.uber.org/zap"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRoutes registers auth routes on the given router group.
// Register and Login are public; Me requires authentication.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	authGroup := rg.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.GET("/me", authMiddleware, h.Me)
}

type registerRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6,max=128"`
	InviteCode string `json:"inviteCode" binding:"required"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	result, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.InviteCode)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": result,
	})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// Me handles GET /api/v1/auth/me.
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.authService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": user,
	})
}

// handleServiceError maps service errors to appropriate HTTP responses.
//
// AppError values carry a structured business error (HTTP status + code +
// message) and are considered expected outcomes, so they are forwarded to
// the client without generating server-side ERROR logs. Any other error is
// treated as unexpected: it is logged at ERROR level with the full wrapped
// chain, request path and method, so that "500 internal server error" is
// never a silent failure in the access log. The request-scoped zap logger
// (carrying requestId) is used when available, otherwise the global logger
// is used as a fallback.
func handleServiceError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{
			"error": gin.H{
				"code":    appErr.Code,
				"message": appErr.Message,
			},
		})
		return
	}
	logger := zap.L()
	if v, exists := c.Get("logger"); exists {
		if lg, ok := v.(*zap.Logger); ok && lg != nil {
			logger = lg
		}
	}
	logger.Error("unhandled service error",
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Error(err),
	)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "internal server error",
		},
	})
}
