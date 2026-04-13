package feedback

import (
	"context"
	"fmt"
	"net/http"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"go.uber.org/zap"
)

const (
	// allowedRatingHelpful is the value for a positive rating.
	allowedRatingHelpful = "helpful"
	// allowedRatingNotHelpful is the value for a negative rating.
	allowedRatingNotHelpful = "not_helpful"
	// maxCommentLength is the maximum length of the comment field.
	maxCommentLength = 500
)

// CreateFeedbackInput carries validated user input for creating a feedback entry.
type CreateFeedbackInput struct {
	AssetAnalysisID int64  `json:"assetAnalysisId"`
	Rating          string `json:"rating"`
	Comment         string `json:"comment,omitempty"`
}

// Service handles feedback business logic.
type Service struct {
	feedbackRepo *repo.UserFeedbackRepo
	logger       *zap.Logger
}

// NewService constructs a feedback Service.
func NewService(feedbackRepo *repo.UserFeedbackRepo, logger *zap.Logger) *Service {
	return &Service{
		feedbackRepo: feedbackRepo,
		logger:       logger,
	}
}

// Create validates the input and persists a new feedback record. Returns the
// generated feedback ID on success.
func (s *Service) Create(ctx context.Context, userID int64, input *CreateFeedbackInput) (int64, error) {
	if input == nil {
		return 0, model.NewAppError(http.StatusBadRequest, "INVALID_INPUT", "input is required")
	}

	if input.Rating != allowedRatingHelpful && input.Rating != allowedRatingNotHelpful {
		return 0, model.NewAppError(http.StatusBadRequest, "INVALID_RATING",
			fmt.Sprintf("rating must be %q or %q", allowedRatingHelpful, allowedRatingNotHelpful))
	}

	if len([]rune(input.Comment)) > maxCommentLength {
		return 0, model.NewAppError(http.StatusBadRequest, "COMMENT_TOO_LONG",
			fmt.Sprintf("comment must not exceed %d characters", maxCommentLength))
	}

	feedbackID, err := s.feedbackRepo.Create(ctx, userID, input.AssetAnalysisID, input.Rating, input.Comment)
	if err != nil {
		return 0, fmt.Errorf("create feedback: %w", err)
	}

	s.logger.Info("user feedback created",
		zap.Int64("user_id", userID),
		zap.Int64("feedback_id", feedbackID),
		zap.String("rating", input.Rating),
	)

	return feedbackID, nil
}
