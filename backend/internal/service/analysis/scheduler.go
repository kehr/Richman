package analysis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/notification/adapter"
	"github.com/richman/backend/internal/repo"
	notificationSvc "github.com/richman/backend/internal/service/notification"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler runs periodic analysis jobs and pushes notifications to users.
type Scheduler struct {
	analysisSvc *Service
	notifSvc    *notificationSvc.Service
	holdingRepo *repo.HoldingRepo
	userRepo    *repo.UserRepo
	logger      *zap.Logger
	cron        *cron.Cron
}

// NewScheduler creates a new analysis Scheduler.
func NewScheduler(
	analysisSvc *Service,
	notifSvc *notificationSvc.Service,
	holdingRepo *repo.HoldingRepo,
	userRepo *repo.UserRepo,
	logger *zap.Logger,
) *Scheduler {
	// Use Asia/Shanghai timezone for all cron schedules.
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Warn("failed to load Asia/Shanghai timezone, using UTC", zap.Error(err))
		return &Scheduler{
			analysisSvc: analysisSvc,
			notifSvc:    notifSvc,
			holdingRepo: holdingRepo,
			userRepo:    userRepo,
			logger:      logger,
			cron:        cron.New(),
		}
	}

	return &Scheduler{
		analysisSvc: analysisSvc,
		notifSvc:    notifSvc,
		holdingRepo: holdingRepo,
		userRepo:    userRepo,
		logger:      logger,
		cron:        cron.New(cron.WithLocation(loc)),
	}
}

// Start registers cron jobs and starts the scheduler.
func (s *Scheduler) Start() {
	// 08:30 CST: A-share AM brief (broad + industry)
	if _, err := s.cron.AddFunc("30 8 * * 1-5", func() {
		s.runJob("am_brief", []string{"a_share_broad", "a_share_industry"})
	}); err != nil {
		s.logger.Error("failed to register am_brief cron job", zap.Error(err))
	}

	// 15:30 CST: A-share + gold PM digest
	if _, err := s.cron.AddFunc("30 15 * * 1-5", func() {
		s.runJob("pm_digest", []string{"a_share_broad", "a_share_industry", "gold_etf"})
	}); err != nil {
		s.logger.Error("failed to register pm_digest cron job", zap.Error(err))
	}

	// 06:00 CST: US stock digest (Tue-Sat because US markets trade Mon-Fri)
	if _, err := s.cron.AddFunc("0 6 * * 2-6", func() {
		s.runJob("us_digest", []string{"us_stock"})
	}); err != nil {
		s.logger.Error("failed to register us_digest cron job", zap.Error(err))
	}

	s.cron.Start()
	s.logger.Info("analysis scheduler started")
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("analysis scheduler stopped")
}

// runJob executes an analysis job for all users with holdings matching the given asset types.
func (s *Scheduler) runJob(messageType string, assetTypes []string) {
	ctx := context.Background()

	s.logger.Info("cron job started",
		zap.String("message_type", messageType),
		zap.Strings("asset_types", assetTypes),
	)

	// Collect all users who have holdings of relevant types.
	userHoldings := make(map[int64][]model.Holding)

	for _, assetType := range assetTypes {
		holdings, err := s.holdingRepo.ListHoldingsByAssetType(ctx, assetType)
		if err != nil {
			s.logger.Error("failed to list holdings by asset type",
				zap.String("asset_type", assetType),
				zap.Error(err),
			)
			continue
		}
		for i := range holdings {
			h := holdings[i]
			userHoldings[h.UserID] = append(userHoldings[h.UserID], h)
		}
	}

	if len(userHoldings) == 0 {
		s.logger.Info("no users with relevant holdings, skipping",
			zap.String("message_type", messageType),
		)
		return
	}

	for userID, holdings := range userHoldings {
		s.processUserJob(ctx, userID, holdings, messageType)
	}

	s.logger.Info("cron job completed",
		zap.String("message_type", messageType),
		zap.Int("user_count", len(userHoldings)),
	)
}

// processUserJob runs analysis on a user's holdings and sends a notification.
func (s *Scheduler) processUserJob(
	ctx context.Context, userID int64,
	holdings []model.Holding, messageType string,
) {
	var cardSummaries []string

	for i := range holdings {
		card, err := s.analysisSvc.AnalyzeHolding(ctx, userID, &holdings[i])
		if err != nil {
			s.logger.Warn("analysis failed for holding in cron job",
				zap.Int64("user_id", userID),
				zap.Int64("holding_id", holdings[i].HoldingID),
				zap.String("asset", holdings[i].AssetCode),
				zap.Error(err),
			)
			continue
		}
		if card != nil {
			summary := fmt.Sprintf("[%s] %s: %s", card.AssetCode, card.AssetName, card.ActionAdvice)
			cardSummaries = append(cardSummaries, summary)
		}
	}

	if len(cardSummaries) == 0 {
		return
	}

	subject := buildSubject(messageType)
	body := strings.Join(cardSummaries, "\n\n")
	cardSummary := cardSummaries[0]
	if len(cardSummaries) > 1 {
		cardSummary = fmt.Sprintf("%s (and %d more)", cardSummaries[0], len(cardSummaries)-1)
	}

	// Get user email for email channel.
	user, err := s.userRepo.GetUserByID(ctx, userID)
	var email string
	if err == nil && user != nil {
		email = user.Email
	}

	msg := adapter.Message{
		Subject:     subject,
		Body:        body,
		CardSummary: cardSummary,
		UserEmail:   email,
	}

	if err := s.notifSvc.SendToUser(ctx, userID, msg, messageType); err != nil {
		s.logger.Warn("failed to send cron notification",
			zap.Int64("user_id", userID),
			zap.String("message_type", messageType),
			zap.Error(err),
		)
	}
}

// buildSubject returns a human-readable subject for the given message type.
func buildSubject(messageType string) string {
	switch messageType {
	case "am_brief":
		return "Morning Market Brief"
	case "pm_digest":
		return "Afternoon Market Digest"
	case "us_digest":
		return "US Market Digest"
	default:
		return "Market Update"
	}
}
