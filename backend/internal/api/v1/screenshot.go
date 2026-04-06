package v1

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/service/screenshot"
)

// ScreenshotHandler exposes the portfolio screenshot recognition endpoint.
type ScreenshotHandler struct {
	service *screenshot.Service
}

// NewScreenshotHandler creates a new ScreenshotHandler.
func NewScreenshotHandler(service *screenshot.Service) *ScreenshotHandler {
	return &ScreenshotHandler{service: service}
}

// RegisterRoutes wires the screenshot routes under the given group.
// All screenshot routes require authentication.
func (h *ScreenshotHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	portfolio := rg.Group("/portfolio", authMiddleware)
	portfolio.POST("/import-screenshot", h.ImportScreenshot)
}

// maxUploadFormBytes bounds the multipart body we are willing to buffer.
// We add a small envelope on top of the service-level cap to allow for
// form headers without masking a correct 413.
const maxUploadFormBytes = screenshot.MaxImageBytes + 64*1024

// ImportScreenshot handles POST /api/v1/portfolio/import-screenshot.
// It accepts a multipart/form-data body with a "file" field holding the
// portfolio screenshot and returns a structured preview for the client
// to confirm. The endpoint does not persist any data.
func (h *ScreenshotHandler) ImportScreenshot(c *gin.Context) {
	userID := middleware.GetUserID(c)

	contentType := c.GetHeader("Content-Type")
	if !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": gin.H{
				"code":    "UNSUPPORTED_MEDIA_TYPE",
				"message": "expected multipart/form-data body",
			},
		})
		return
	}

	// Bound the request body early so a hostile client cannot force us to
	// buffer gigabytes before we reach the service layer.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadFormBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "missing 'file' field in multipart body",
			},
		})
		return
	}

	// Fast-fail on size before we actually read the file.
	if fileHeader.Size > screenshot.MaxImageBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": gin.H{
				"code":    "FILE_TOO_LARGE",
				"message": "image must be no larger than 5 MB",
			},
		})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "failed to read uploaded file",
			},
		})
		return
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "failed to read uploaded file",
			},
		})
		return
	}

	mime := fileHeader.Header.Get("Content-Type")
	if mime == "" {
		mime = http.DetectContentType(data)
	}

	resp, err := h.service.Recognize(c.Request.Context(), userID, screenshot.RecognizeRequest{
		ImageData: data,
		ImageMIME: mime,
	})
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
