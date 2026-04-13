package model

import "time"

// UserFeedback maps to rm_user_feedback. Users can rate and comment on
// analysis quality after viewing an asset analysis.
type UserFeedback struct {
	FeedbackID      int64     `json:"feedbackId"`
	UserID          int64     `json:"userId"`
	AssetAnalysisID int64     `json:"assetAnalysisId"`
	Rating          string    `json:"rating"`
	Comment         *string   `json:"comment,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
