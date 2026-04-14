package v1

import (
	"errors"
	"net/http"
	"sync"
	"time"

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
// Register and Login are public with IP rate limiting (5 req/min);
// Me and DeleteAccount require authentication.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	authGroup := rg.Group("/auth")
	authRL := authIPRateLimit(5, time.Minute)
	authGroup.POST("/register", authRL, h.Register)
	authGroup.POST("/login", authRL, h.Login)
	authGroup.GET("/me", authMiddleware, h.Me)
	authGroup.DELETE("/account", authMiddleware, h.DeleteAccount)
}

// authIPRateLimit is a lightweight in-process sliding-window rate limiter for
// the auth endpoints. Separate from the v2 middleware package to avoid import
// cycles while reusing the same algorithm.
func authIPRateLimit(maxPerWindow int, window time.Duration) gin.HandlerFunc {
	type entry struct {
		mu         sync.Mutex
		timestamps []time.Time
	}
	var (
		mu      sync.Mutex
		entries = make(map[string]*entry)
	)
	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		e, ok := entries[ip]
		if !ok {
			e = &entry{}
			entries[ip] = e
		}
		mu.Unlock()

		e.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-window)
		valid := e.timestamps[:0]
		for _, ts := range e.timestamps {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}
		e.timestamps = valid

		if len(e.timestamps) >= maxPerWindow {
			e.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "too many requests",
				},
			})
			return
		}
		e.timestamps = append(e.timestamps, now)
		e.mu.Unlock()

		c.Next()
	}
}

// Password length is enforced by the binding tag (8-128). Character class
// complexity (upper + lower + digit) is validated in the service layer so
// the same rule applies to future ChangePassword flows (richman TRD SS22.5).
type registerRequest struct {
	Email              string `json:"email" binding:"required,email"`
	Password           string `json:"password" binding:"required,min=8,max=128"`
	InviteCode         string `json:"inviteCode" binding:"required"`
	DisclaimerAccepted bool   `json:"disclaimerAccepted"`
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

	result, err := h.authService.Register(
		c.Request.Context(), req.Email, req.Password, req.InviteCode, req.DisclaimerAccepted,
	)
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

// deleteAccountRequest is the request body for DELETE /api/v1/auth/account.
type deleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

// DeleteAccount handles DELETE /api/v1/auth/account.
// Requires the authenticated user to confirm their password. On success the
// user record is soft-deleted and the client must discard the JWT token.
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req deleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	if err := h.authService.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"message": "account deleted"},
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
