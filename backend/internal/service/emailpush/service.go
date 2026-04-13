package emailpush

import (
	"context"
	"fmt"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/richson"
	emailtemplate "github.com/richman/backend/internal/service/emailpush/template"
	"go.uber.org/zap"
)

const (
	// maxPushesPerDay is the maximum number of push emails a single user can
	// receive in a calendar day. Priority: daily briefing > holding suggestion >
	// market alert > weekly insight.
	maxPushesPerDay = 3

	// goldAssetCode is the ticker used to query gold ETF analyses.
	goldAssetCode = "GLD"

	// userPageSize is the cursor page size for iterating all users.
	userPageSize = 200

	// channelTypeEmail identifies the email channel in rm_notification_logs.
	channelTypeEmail = "email"

	// unsubscribePathTemplate is the path appended to the app base URL for the
	// unsubscribe link injected into every email footer.
	unsubscribePathTemplate = "/settings"

	// disclaimerZH is the legal disclaimer in Chinese.
	disclaimerZH = "Richman 提供的所有信息仅供参考，不构成投资建议。投资有风险，入市需谨慎。"
	// disclaimerEN is the legal disclaimer in English.
	disclaimerEN = "All information provided by Richman is for reference only and does not constitute investment advice. Investment involves risk."
)

// Service orchestrates the four email push workflows: daily briefing, weekly
// insight, market alert, and holding suggestion.
type Service struct {
	userRepo       *repo.UserRepo
	analysisRepo   *repo.AssetAnalysisReadRepo
	holdingRepo    *repo.HoldingRepo
	cardRepo       *repo.DecisionCardRepo
	eventAlertRepo *repo.EventAlertReadRepo
	richsonClient  *richson.Client
	sender         *Sender
	engine         *emailtemplate.Engine
	notifLogRepo   *repo.NotificationLogRepo
	appBaseURL     string
	logger         *zap.Logger
}

// NewService creates a new EmailPushService. appBaseURL is the public-facing
// root URL (e.g. "https://richman.app") used to construct unsubscribe links.
func NewService(
	userRepo *repo.UserRepo,
	analysisRepo *repo.AssetAnalysisReadRepo,
	holdingRepo *repo.HoldingRepo,
	cardRepo *repo.DecisionCardRepo,
	eventAlertRepo *repo.EventAlertReadRepo,
	richsonClient *richson.Client,
	sender *Sender,
	engine *emailtemplate.Engine,
	notifLogRepo *repo.NotificationLogRepo,
	appBaseURL string,
	logger *zap.Logger,
) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		userRepo:       userRepo,
		analysisRepo:   analysisRepo,
		holdingRepo:    holdingRepo,
		cardRepo:       cardRepo,
		eventAlertRepo: eventAlertRepo,
		richsonClient:  richsonClient,
		sender:         sender,
		engine:         engine,
		notifLogRepo:   notifLogRepo,
		appBaseURL:     appBaseURL,
		logger:         logger,
	}
}

// ---- Push frequency control ----

// checkAndLogPush verifies that the user has not exceeded maxPushesPerDay,
// then records a new notification log entry. Returns false if the push should
// be skipped.
func (s *Service) checkAndLogPush(ctx context.Context, userID int64, messageType string) (bool, error) {
	count, err := s.notifLogRepo.CountTodayByUser(ctx, userID, channelTypeEmail)
	if err != nil {
		return false, fmt.Errorf("count today pushes: %w", err)
	}
	if count >= maxPushesPerDay {
		s.logger.Warn("daily push limit reached, skipping",
			zap.Int64("user_id", userID),
			zap.String("message_type", messageType),
			zap.Int("count", count),
		)
		return false, nil
	}

	if _, err := s.notifLogRepo.Create(ctx, userID, channelTypeEmail, messageType, "sent", ""); err != nil {
		// Log but do not block the send; a missed log is not worth skipping delivery.
		s.logger.Error("create notification log failed",
			zap.Int64("user_id", userID),
			zap.String("message_type", messageType),
			zap.Error(err),
		)
	}
	return true, nil
}

// ---- Helpers ----

func (s *Service) unsubscribeURL() string {
	return s.appBaseURL + unsubscribePathTemplate
}

// ---- 1. SendDailyBriefing ----

// SendDailyBriefing sends morning briefing emails to all users who have
// email_push_enabled=TRUE. It fetches market data in parallel, then iterates
// users via cursor pagination (200 per page) sending personalized emails.
// Content is assembled from pre-computed data; no LLM calls are made.
func (s *Service) SendDailyBriefing(ctx context.Context) error {
	s.logger.Info("starting daily briefing push")

	// Parallel data fetch.
	type parallelResult struct {
		regime      *richson.MarketRegimeResponse
		goldLatest  *model.AssetAnalysis
		goldPrev    *model.AssetAnalysis
		events      *richson.EventsRadarResponse
		regimeErr   error
		goldLatErr  error
		goldPrevErr error
		eventsErr   error
	}

	resCh := make(chan parallelResult, 1)
	go func() {
		var r parallelResult
		r.regime, r.regimeErr = s.richsonClient.GetMarketRegime(ctx)
		r.goldLatest, r.goldLatErr = s.analysisRepo.GetLatestByAssetCode(ctx, goldAssetCode)
		r.events, r.eventsErr = s.richsonClient.GetEventsRadar(ctx)

		// Fetch yesterday's gold analysis: second latest row.
		if r.goldLatest != nil {
			r.goldPrev, r.goldPrevErr = s.analysisRepo.GetSecondLatestByAssetCode(ctx, goldAssetCode, r.goldLatest.AssetAnalysisID)
		}

		resCh <- r
	}()

	r := <-resCh

	if r.regimeErr != nil {
		s.logger.Warn("failed to fetch market regime for daily briefing", zap.Error(r.regimeErr))
	}
	if r.goldLatErr != nil {
		s.logger.Warn("failed to fetch gold analysis for daily briefing", zap.Error(r.goldLatErr))
	}
	if r.eventsErr != nil {
		s.logger.Warn("failed to fetch events for daily briefing", zap.Error(r.eventsErr))
	}

	// Build event summaries.
	var events []emailtemplate.EventSummary
	if r.events != nil {
		for _, ev := range r.events.Events {
			events = append(events, emailtemplate.EventSummary{
				Title:         ev.Title,
				GoldDirection: ev.GoldDirection,
				Impact:        ev.Impact,
			})
		}
	}

	// Compute gold score delta.
	var goldScore float64
	var goldDelta float64
	if r.goldLatest != nil {
		goldScore = r.goldLatest.OverallScore
		if r.goldPrev != nil {
			goldDelta = r.goldLatest.OverallScore - r.goldPrev.OverallScore
		}
	}

	// Regime label.
	regimeLabel := ""
	if r.regime != nil {
		regimeLabel = r.regime.RegimeLabel
	}

	// Cursor pagination over users.
	var lastID int64 = 0
	totalSent := 0
	totalSkipped := 0

	for {
		users, err := s.userRepo.ListEmailPushEnabled(ctx, lastID, userPageSize)
		if err != nil {
			return fmt.Errorf("list email push users: %w", err)
		}
		if len(users) == 0 {
			break
		}

		for _, u := range users {
			if err := s.sendDailyBriefingToUser(ctx, &u, regimeLabel, goldScore, goldDelta, events); err != nil {
				s.logger.Error("daily briefing to user failed",
					zap.Int64("user_id", u.UserID),
					zap.Error(err),
				)
				totalSkipped++
			} else {
				totalSent++
			}
		}

		if len(users) < userPageSize {
			break
		}
		lastID = users[len(users)-1].UserID
	}

	s.logger.Info("daily briefing push completed",
		zap.Int("sent", totalSent),
		zap.Int("skipped", totalSkipped),
	)
	return nil
}

// sendDailyBriefingToUser delivers the daily briefing to a single user.
func (s *Service) sendDailyBriefingToUser(
	ctx context.Context,
	u *model.User,
	regimeLabel string,
	goldScore, goldDelta float64,
	events []emailtemplate.EventSummary,
) error {
	ok, err := s.checkAndLogPush(ctx, u.UserID, "daily_briefing")
	if err != nil || !ok {
		return err
	}

	// Fetch holdings and latest decision cards.
	holdings, err := s.holdingRepo.ListHoldingsByUser(ctx, u.UserID)
	if err != nil {
		s.logger.Warn("failed to fetch holdings for daily briefing",
			zap.Int64("user_id", u.UserID),
			zap.Error(err),
		)
	}

	var holdingSummaries []emailtemplate.HoldingSummary
	if len(holdings) > 0 {
		holdingIDs := make([]int64, len(holdings))
		for i, h := range holdings {
			holdingIDs[i] = h.HoldingID
		}
		cards, err := s.cardRepo.GetLatestByHoldings(ctx, holdingIDs)
		if err != nil {
			s.logger.Warn("failed to fetch cards for daily briefing",
				zap.Int64("user_id", u.UserID),
				zap.Error(err),
			)
		}
		for _, h := range holdings {
			hs := emailtemplate.HoldingSummary{
				AssetName: h.AssetName,
				AssetCode: h.AssetCode,
			}
			if card, ok := cards[h.HoldingID]; ok {
				hs.ActionAdvice = card.ActionAdvice
				hs.Confidence = card.Confidence
			}
			holdingSummaries = append(holdingSummaries, hs)
		}
	}

	locale := u.Language
	templateName := "daily_briefing_" + locale
	disclaimer := disclaimerEN
	if locale == model.LanguageZH {
		disclaimer = disclaimerZH
	}

	data := emailtemplate.DailyBriefingData{
		RegimeLabel:    regimeLabel,
		GoldScore:      goldScore,
		GoldScoreDelta: goldDelta,
		Events:         events,
		Holdings:       holdingSummaries,
		HasHoldings:    len(holdingSummaries) > 0,
		UnsubscribeURL: s.unsubscribeURL(),
		Disclaimer:     disclaimer,
	}

	html, err := s.engine.Render(templateName, data)
	if err != nil {
		// Fall back to English template if locale-specific one is missing.
		s.logger.Warn("template render failed, falling back to en",
			zap.String("template", templateName),
			zap.Error(err),
		)
		html, err = s.engine.Render("daily_briefing_en", data)
		if err != nil {
			return fmt.Errorf("render daily_briefing template: %w", err)
		}
	}

	subject := "Richman Daily Briefing"
	if locale == model.LanguageZH {
		subject = "Richman 每日简报"
	}

	return s.sender.Send(ctx, u.Email, subject, html)
}

// ---- 2. SendWeeklyInsight ----

// SendWeeklyInsight fetches a weekly market insight from richson and broadcasts
// it to all email-push-enabled users. On richson failure the push is skipped
// with an ERROR log; the failure is not propagated to callers.
func (s *Service) SendWeeklyInsight(ctx context.Context) error {
	s.logger.Info("starting weekly insight push")

	// Fetch insight for each locale and send to matching users.
	for _, locale := range []string{model.LanguageZH, model.LanguageEN} {
		if err := s.sendWeeklyInsightForLocale(ctx, locale); err != nil {
			s.logger.Error("weekly insight for locale failed",
				zap.String("locale", locale),
				zap.Error(err),
			)
			// Per SS7.6: on failure, skip (don't degrade), log ERROR.
		}
	}

	return nil
}

// sendWeeklyInsightForLocale handles one language variant of the weekly insight.
func (s *Service) sendWeeklyInsightForLocale(ctx context.Context, locale string) error {
	resp, err := s.richsonClient.GenerateWeeklyInsight(ctx, richson.WeeklyInsightRequest{
		Locale: locale,
	})
	if err != nil {
		s.logger.Error("richson weekly insight failed",
			zap.String("locale", locale),
			zap.Error(err),
		)
		return nil // skip, do not propagate
	}

	sections := make([]emailtemplate.InsightSection, len(resp.Sections))
	for i, sec := range resp.Sections {
		sections[i] = emailtemplate.InsightSection{
			Heading: sec.Heading,
			Content: sec.Content,
		}
	}

	disclaimer := disclaimerEN
	if locale == model.LanguageZH {
		disclaimer = disclaimerZH
	}

	data := emailtemplate.WeeklyInsightData{
		Title:          resp.Title,
		Sections:       sections,
		UnsubscribeURL: s.unsubscribeURL(),
		Disclaimer:     disclaimer,
	}

	html, err := s.engine.Render("weekly_insight_"+locale, data)
	if err != nil {
		return fmt.Errorf("render weekly_insight_%s: %w", locale, err)
	}

	subject := "Richman Weekly Insight"
	if locale == model.LanguageZH {
		subject = "Richman 每周洞察"
	}

	// Collect all email-push-enabled users for this locale.
	var lastID int64 = 0
	var recipients []string
	for {
		users, err := s.userRepo.ListEmailPushEnabledByLocale(ctx, locale, lastID, userPageSize)
		if err != nil {
			return fmt.Errorf("list users by locale: %w", err)
		}
		for _, u := range users {
			recipients = append(recipients, u.Email)
		}
		if len(users) < userPageSize {
			break
		}
		lastID = users[len(users)-1].UserID
	}

	if len(recipients) == 0 {
		s.logger.Info("no recipients for weekly insight", zap.String("locale", locale))
		return nil
	}

	return s.sender.SendBatch(ctx, recipients, subject, html)
}

// ---- 3. SendMarketAlert ----

// SendMarketAlert sends an event-driven alert to all email-push-enabled users
// when a Polymarket event probability changes by more than the alert threshold.
func (s *Service) SendMarketAlert(ctx context.Context, alert *model.EventAlert) error {
	s.logger.Info("sending market alert",
		zap.String("event_slug", alert.EventSlug),
		zap.Float64("delta", alert.Delta),
	)

	goldDir := ""
	if alert.GoldDirection != nil {
		goldDir = *alert.GoldDirection
	}

	var lastID int64 = 0
	totalSent := 0

	for {
		users, err := s.userRepo.ListEmailPushEnabled(ctx, lastID, userPageSize)
		if err != nil {
			return fmt.Errorf("list email push users for alert: %w", err)
		}
		if len(users) == 0 {
			break
		}

		for _, u := range users {
			if err := s.sendMarketAlertToUser(ctx, &u, alert, goldDir); err != nil {
				s.logger.Error("market alert to user failed",
					zap.Int64("user_id", u.UserID),
					zap.Error(err),
				)
			} else {
				totalSent++
			}
		}

		if len(users) < userPageSize {
			break
		}
		lastID = users[len(users)-1].UserID
	}

	s.logger.Info("market alert push completed",
		zap.String("event_slug", alert.EventSlug),
		zap.Int("sent", totalSent),
	)
	return nil
}

// sendMarketAlertToUser delivers the market alert to a single user.
func (s *Service) sendMarketAlertToUser(
	ctx context.Context,
	u *model.User,
	alert *model.EventAlert,
	goldDir string,
) error {
	ok, err := s.checkAndLogPush(ctx, u.UserID, "market_alert")
	if err != nil || !ok {
		return err
	}

	locale := u.Language
	disclaimer := disclaimerEN
	if locale == model.LanguageZH {
		disclaimer = disclaimerZH
	}

	data := emailtemplate.MarketAlertData{
		EventTitle:      alert.EventTitle,
		PrevProbability: alert.PrevProbability,
		CurrProbability: alert.CurrProbability,
		Delta:           alert.Delta,
		GoldDirection:   goldDir,
		UnsubscribeURL:  s.unsubscribeURL(),
		Disclaimer:      disclaimer,
	}

	templateName := "market_alert_" + locale
	html, err := s.engine.Render(templateName, data)
	if err != nil {
		html, err = s.engine.Render("market_alert_en", data)
		if err != nil {
			return fmt.Errorf("render market_alert template: %w", err)
		}
	}

	subject := "Richman Market Alert"
	if locale == model.LanguageZH {
		subject = "Richman 市场预警"
	}

	return s.sender.Send(ctx, u.Email, subject, html)
}

// ---- 4. SendHoldingSuggestion ----

// SendHoldingSuggestion sends a personalized holding suggestion email to a
// single user after a holding analysis completes.
func (s *Service) SendHoldingSuggestion(ctx context.Context, userID int64, card *model.DecisionCard) error {
	u, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user for holding suggestion: %w", err)
	}
	if u == nil {
		return fmt.Errorf("user %d not found", userID)
	}

	ok, err := s.checkAndLogPush(ctx, u.UserID, "holding_suggestion")
	if err != nil || !ok {
		return err
	}

	locale := u.Language
	disclaimer := disclaimerEN
	if locale == model.LanguageZH {
		disclaimer = disclaimerZH
	}

	data := emailtemplate.HoldingSuggestionData{
		AssetName:      card.AssetName,
		AssetCode:      card.AssetCode,
		ActionAdvice:   card.ActionAdvice,
		DetailedAdvice: card.DetailedAdvice,
		RiskWarnings:   card.RiskWarnings,
		Confidence:     card.Confidence,
		UnsubscribeURL: s.unsubscribeURL(),
		Disclaimer:     disclaimer,
	}

	templateName := "holding_suggestion_" + locale
	html, err := s.engine.Render(templateName, data)
	if err != nil {
		html, err = s.engine.Render("holding_suggestion_en", data)
		if err != nil {
			return fmt.Errorf("render holding_suggestion template: %w", err)
		}
	}

	subject := fmt.Sprintf("Richman - %s Suggestion", card.AssetName)
	if locale == model.LanguageZH {
		subject = fmt.Sprintf("Richman - %s 持仓建议", card.AssetName)
	}

	s.logger.Info("sending holding suggestion",
		zap.Int64("user_id", userID),
		zap.String("asset_code", card.AssetCode),
	)
	return s.sender.Send(ctx, u.Email, subject, html)
}

