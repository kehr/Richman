package analysis

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/richman/backend/internal/datasource"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/notification/adapter"
	"github.com/richman/backend/internal/repo"
	notificationSvc "github.com/richman/backend/internal/service/notification"
	scheduleSvc "github.com/richman/backend/internal/service/schedule"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// scheduleServiceDep is the subset of schedule.Service used by the Scheduler.
// Defined as a local interface to avoid forcing callers to import the schedule
// package, and to simplify testing.
type scheduleServiceDep interface {
	ListActiveUserScheduleSettings(ctx context.Context) ([]model.UserScheduleSettings, error)
	GetUserScheduleSettings(ctx context.Context, userID int64) (*model.UserScheduleSettings, error)
	ComputeNextAnalysisAt(
		ctx context.Context,
		userID, holdingID int64,
		market string,
		lastAnalyzedAt *time.Time,
		now time.Time,
	) (time.Time, error)
}

// windowKind identifies a trading window within a market.
type windowKind string

const (
	windowPre  windowKind = "pre"
	windowPost windowKind = "post"
)

// userEntryKey uniquely identifies a cron entry registered for a user.
type userEntryKey struct {
	userID int64
	market string
	window windowKind
}

// Scheduler runs periodic analysis jobs and pushes notifications to users.
type Scheduler struct {
	analysisSvc *Service
	notifSvc    *notificationSvc.Service
	holdingRepo *repo.HoldingRepo
	cardRepo    *repo.DecisionCardRepo
	userRepo    *repo.UserRepo
	schedSvc    scheduleServiceDep
	logger      *zap.Logger
	cron        *cron.Cron
	loc         *time.Location

	// mu guards entryIDs.
	mu       sync.Mutex
	entryIDs map[userEntryKey]cron.EntryID

	// dstEntryID tracks the active one-shot DST callback entry.
	dstEntryID cron.EntryID
}

// NewScheduler creates a new analysis Scheduler.
func NewScheduler(
	analysisSvc *Service,
	notifSvc *notificationSvc.Service,
	holdingRepo *repo.HoldingRepo,
	cardRepo *repo.DecisionCardRepo,
	userRepo *repo.UserRepo,
	schedSvc scheduleServiceDep,
	logger *zap.Logger,
) *Scheduler {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Warn("failed to load Asia/Shanghai timezone, using UTC", zap.Error(err))
		loc = time.UTC
	}

	return &Scheduler{
		analysisSvc: analysisSvc,
		notifSvc:    notifSvc,
		holdingRepo: holdingRepo,
		cardRepo:    cardRepo,
		userRepo:    userRepo,
		schedSvc:    schedSvc,
		logger:      logger,
		loc:         loc,
		cron:        cron.New(cron.WithLocation(loc)),
		entryIDs:    make(map[userEntryKey]cron.EntryID),
	}
}

// Start loads all user schedule configs, registers per-user cron entries, and
// starts the scheduler. Users with a saved settings row use that config; users
// with active holdings but no saved row receive system-default entries.
func (s *Scheduler) Start() {
	ctx := context.Background()

	// Load users who have explicitly saved schedule settings.
	settings, err := s.schedSvc.ListActiveUserScheduleSettings(ctx)
	if err != nil {
		s.logger.Error("failed to load user schedule settings on startup", zap.Error(err))
	}

	coveredUsers := make(map[int64]struct{}, len(settings))
	for i := range settings {
		s.registerUserEntries(&settings[i])
		coveredUsers[settings[i].UserID] = struct{}{}
	}

	// Register system-default entries for users who have holdings but no saved
	// settings row. GetUserScheduleSettings returns defaults when no row exists.
	allUsersWithHoldings, err := s.holdingRepo.ListUsersWithHoldings(ctx)
	if err != nil {
		s.logger.Error("failed to list users with holdings on startup", zap.Error(err))
	}

	defaultCount := 0
	for _, userID := range allUsersWithHoldings {
		if _, covered := coveredUsers[userID]; covered {
			continue
		}
		defaults, err := s.schedSvc.GetUserScheduleSettings(ctx, userID)
		if err != nil {
			s.logger.Error("failed to get default schedule settings for user",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			continue
		}
		s.registerUserEntries(defaults)
		defaultCount++
	}

	s.scheduleDSTCallback()

	s.cron.Start()
	s.logger.Info("analysis scheduler started",
		zap.Int("configured_users", len(settings)),
		zap.Int("default_users", defaultCount),
	)
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("analysis scheduler stopped")
}

// ReloadUser removes all existing cron entries for the given user and
// re-registers them from current schedule settings. Implements the
// v1.ScheduleReloader interface so the schedule HTTP handler can trigger a
// live reload after a settings save.
func (s *Scheduler) ReloadUser(ctx context.Context, userID int64) error {
	s.removeUserEntries(userID)

	settings, err := s.schedSvc.GetUserScheduleSettings(ctx, userID)
	if err != nil {
		s.logger.Error("failed to reload user schedule settings",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	s.registerUserEntries(settings)
	s.logger.Info("reloaded schedule entries for user", zap.Int64("user_id", userID))
	return nil
}

// removeUserEntries removes all cron entries for a user.
func (s *Scheduler) removeUserEntries(userID int64) {
	s.mu.Lock()
	var toRemove []cron.EntryID
	for key, id := range s.entryIDs {
		if key.userID == userID {
			toRemove = append(toRemove, id)
			delete(s.entryIDs, key)
		}
	}
	s.mu.Unlock()

	for _, id := range toRemove {
		s.cron.Remove(id)
	}
}

// removeUserUSEntries removes only the US-market entries for a user.
func (s *Scheduler) removeUserUSEntries(userID int64) {
	s.mu.Lock()
	var toRemove []cron.EntryID
	for key, id := range s.entryIDs {
		if key.userID == userID && key.market == "us" {
			toRemove = append(toRemove, id)
			delete(s.entryIDs, key)
		}
	}
	s.mu.Unlock()

	for _, id := range toRemove {
		s.cron.Remove(id)
	}
}

// registerUserEntries creates cron entries for each enabled window in the
// user's settings.
func (s *Scheduler) registerUserEntries(settings *model.UserScheduleSettings) {
	userID := settings.UserID

	if settings.ASharePreEnabled {
		s.registerWindowEntry(userID, "a_share", windowPre, settings.ASharePreTime,
			"1-5",
			[]string{model.AssetTypeAShareBroad, model.AssetTypeAShareIndustry},
			"am_brief",
		)
	}

	// Post-market A-share window also covers gold_etf.
	if settings.ASharePostEnabled {
		s.registerWindowEntry(userID, "a_share", windowPost, settings.ASharePostTime,
			"1-5",
			[]string{model.AssetTypeAShareBroad, model.AssetTypeAShareIndustry, model.AssetTypeGoldETF},
			"pm_digest",
		)
	}

	// US pre-market (disabled by default, honored when user enables it).
	// Tue-Sat because US Mon-Fri translates to Tue-Sat in Asia/Shanghai.
	if settings.USPreEnabled {
		s.registerWindowEntry(userID, "us", windowPre, settings.USPreTime,
			"2-6",
			[]string{model.AssetTypeUSStock},
			"us_pre_brief",
		)
	}

	if settings.USPostEnabled {
		s.registerWindowEntry(userID, "us", windowPost, settings.USPostTime,
			"2-6",
			[]string{model.AssetTypeUSStock},
			"us_digest",
		)
	}
}

// registerUserUSEntries re-registers only the US-market window entries.
func (s *Scheduler) registerUserUSEntries(settings *model.UserScheduleSettings) {
	if settings.USPreEnabled {
		s.registerWindowEntry(settings.UserID, "us", windowPre, settings.USPreTime,
			"2-6", []string{model.AssetTypeUSStock}, "us_pre_brief",
		)
	}
	if settings.USPostEnabled {
		s.registerWindowEntry(settings.UserID, "us", windowPost, settings.USPostTime,
			"2-6", []string{model.AssetTypeUSStock}, "us_digest",
		)
	}
}

// fetcher exposes the datasource.Fetcher from the analysis service so the
// scheduler can fetch recent price history for pre-window context.
func (s *Scheduler) fetcher() *datasource.Fetcher {
	return s.analysisSvc.fetcher
}

// registerWindowEntry parses "HH:MM", builds a cron spec, and registers the
// entry, recording its EntryID.
func (s *Scheduler) registerWindowEntry(
	userID int64,
	market string,
	window windowKind,
	timeStr string,
	weekdays string,
	assetTypes []string,
	messageType string,
) {
	hour, minute, ok := parseHHMM(timeStr)
	if !ok {
		s.logger.Error("invalid window time, skipping entry",
			zap.Int64("user_id", userID),
			zap.String("market", market),
			zap.String("window", string(window)),
			zap.String("time", timeStr),
		)
		return
	}

	spec := fmt.Sprintf("%d %d * * %s", minute, hour, weekdays)
	isPreWindow := window == windowPre

	id, err := s.cron.AddFunc(spec, func() {
		s.runJobForUser(userID, assetTypes, messageType, isPreWindow)
	})
	if err != nil {
		s.logger.Error("failed to register cron entry",
			zap.Int64("user_id", userID),
			zap.String("spec", spec),
			zap.Error(err),
		)
		return
	}

	key := userEntryKey{userID: userID, market: market, window: window}

	// Hold the lock across the entire read+remove+write sequence to eliminate
	// the TOCTOU window. cron.Remove and cron.AddFunc use their own internal
	// lock, so calling them while holding s.mu is safe.
	s.mu.Lock()
	if oldID, exists := s.entryIDs[key]; exists {
		s.cron.Remove(oldID)
	}
	s.entryIDs[key] = id
	s.mu.Unlock()

	s.logger.Debug("registered cron entry",
		zap.Int64("user_id", userID),
		zap.String("market", market),
		zap.String("window", string(window)),
		zap.String("spec", spec),
		zap.Int("entry_id", int(id)),
	)
}

// parseHHMM parses "HH:MM" into (hour, minute, ok).
func parseHHMM(t string) (hour, minute int, ok bool) {
	if len(t) != 5 || t[2] != ':' {
		return 0, 0, false
	}
	var h, m int
	if _, err := fmt.Sscanf(t, "%d:%d", &h, &m); err != nil {
		return 0, 0, false
	}
	if h > 23 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// oneShotSchedule implements cron.Schedule to fire exactly once at a specific
// UTC instant and never repeat (Next returns zero time after the instant).
type oneShotSchedule struct {
	at time.Time
}

func (o oneShotSchedule) Next(now time.Time) time.Time {
	if now.Before(o.at) {
		return o.at
	}
	return time.Time{}
}

// scheduleDSTCallback registers (or replaces) a one-shot cron entry that fires
// at the next EDT/EST transition.
func (s *Scheduler) scheduleDSTCallback() {
	now := time.Now().UTC()
	next := scheduleSvc.NextDSTTransition(now)

	if s.dstEntryID != 0 {
		s.cron.Remove(s.dstEntryID)
	}

	id := s.cron.Schedule(oneShotSchedule{at: next}, cron.FuncJob(func() {
		s.onDSTTransition()
	}))
	s.dstEntryID = id

	s.logger.Info("DST transition callback scheduled",
		zap.Time("fires_at", next),
		zap.Int("entry_id", int(id)),
	)
}

// onDSTTransition is invoked at each EDT/EST boundary. It updates non-custom
// US window times for all configured users and reschedules itself.
func (s *Scheduler) onDSTTransition() {
	now := time.Now()
	isEDT := scheduleSvc.IsEDT(now)

	s.logger.Info("DST transition fired",
		zap.Bool("is_edt", isEDT),
		zap.Time("at", now),
	)

	ctx := context.Background()
	settings, err := s.schedSvc.ListActiveUserScheduleSettings(ctx)
	if err != nil {
		s.logger.Error("DST transition: failed to list user settings", zap.Error(err))
	} else {
		for i := range settings {
			st := &settings[i]
			updated := false

			if !st.USPreCustom {
				if isEDT {
					st.USPreTime = scheduleSvc.DefaultUSPreTimeEDT
				} else {
					st.USPreTime = scheduleSvc.DefaultUSPreTimeEST
				}
				updated = true
			}
			if !st.USPostCustom {
				if isEDT {
					st.USPostTime = scheduleSvc.DefaultUSPostTimeEDT
				} else {
					st.USPostTime = scheduleSvc.DefaultUSPostTimeEST
				}
				updated = true
			}

			if updated {
				s.removeUserUSEntries(st.UserID)
				s.registerUserUSEntries(st)
				s.logger.Info("DST transition: updated US window times",
					zap.Int64("user_id", st.UserID),
					zap.Bool("is_edt", isEDT),
				)
			}
		}
	}

	s.scheduleDSTCallback()
}

// runJobForUser executes an analysis job for a single user's holdings of the
// given asset types, applying a frequency gate before running analysis.
// isPreWindow indicates this is a pre-market trigger, enabling price-delta
// context injection into the synthesis prompt.
func (s *Scheduler) runJobForUser(userID int64, assetTypes []string, messageType string, isPreWindow bool) {
	ctx := context.Background()

	s.logger.Info("cron job triggered for user",
		zap.Int64("user_id", userID),
		zap.String("message_type", messageType),
		zap.Bool("pre_window", isPreWindow),
	)

	// Fetch all user holdings and filter to the relevant asset types.
	allHoldings, err := s.holdingRepo.ListHoldingsByUser(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list holdings for user",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	assetTypeSet := make(map[string]struct{}, len(assetTypes))
	for _, at := range assetTypes {
		assetTypeSet[at] = struct{}{}
	}

	var holdings []model.Holding
	for i := range allHoldings {
		if _, ok := assetTypeSet[allHoldings[i].AssetType]; ok {
			holdings = append(holdings, allHoldings[i])
		}
	}

	if len(holdings) == 0 {
		s.logger.Debug("no holdings of relevant asset types for user, skipping",
			zap.Int64("user_id", userID),
			zap.String("message_type", messageType),
		)
		return
	}

	now := time.Now().In(s.loc)
	eligible := s.filterEligibleHoldings(ctx, userID, holdings, now)
	if len(eligible) == 0 {
		s.logger.Debug("all holdings within frequency gate, skipping",
			zap.Int64("user_id", userID),
			zap.String("message_type", messageType),
		)
		return
	}

	s.processUserJob(ctx, userID, eligible, messageType, isPreWindow)
}

// filterEligibleHoldings returns only holdings for which the frequency minimum
// interval has elapsed since the last analysis.
func (s *Scheduler) filterEligibleHoldings(
	ctx context.Context,
	userID int64,
	holdings []model.Holding,
	now time.Time,
) []model.Holding {
	var eligible []model.Holding

	for i := range holdings {
		h := &holdings[i]
		market := assetTypeToMarket(h.AssetType)

		card, err := s.cardRepo.GetLatestByHolding(ctx, h.HoldingID)
		if err != nil {
			s.logger.Warn("failed to fetch latest card for frequency check, including holding",
				zap.Int64("user_id", userID),
				zap.Int64("holding_id", h.HoldingID),
				zap.Error(err),
			)
			eligible = append(eligible, *h)
			continue
		}

		var lastAnalyzedAt *time.Time
		if card != nil {
			t := card.AnalyzedAt
			lastAnalyzedAt = &t
		}

		nextAt, err := s.schedSvc.ComputeNextAnalysisAt(ctx, userID, h.HoldingID, market, lastAnalyzedAt, now)
		if err != nil {
			s.logger.Warn("failed to compute next analysis time, including holding",
				zap.Int64("user_id", userID),
				zap.Int64("holding_id", h.HoldingID),
				zap.Error(err),
			)
			eligible = append(eligible, *h)
			continue
		}

		if !nextAt.After(now) {
			eligible = append(eligible, *h)
		}
	}

	return eligible
}

// assetTypeToMarket maps asset type strings to schedule market identifiers.
func assetTypeToMarket(assetType string) string {
	if assetType == model.AssetTypeUSStock {
		return scheduleSvc.MarketUSStock
	}
	return scheduleSvc.MarketAShare
}

// processUserJob runs analysis on a user's holdings and sends a notification.
// When isPreWindow is true, per-holding price delta since last analysis is
// fetched and injected into the synthesis prompt as additional context.
func (s *Scheduler) processUserJob(
	ctx context.Context, userID int64,
	holdings []model.Holding, messageType string, isPreWindow bool,
) {
	var cardSummaries []string

	for i := range holdings {
		priceDeltaCtx := ""
		if isPreWindow {
			priceDeltaCtx = s.buildPriceDeltaContext(ctx, &holdings[i])
		}

		card, err := s.analysisSvc.analyzeHolding(ctx, userID, &holdings[i], priceDeltaCtx, "")
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

// buildPriceDeltaContext fetches recent OHLCV data for a holding and returns a
// formatted string describing price changes since the last analysis. Returns an
// empty string if the last card is unavailable, the price fetch fails, or there
// are no new data points since the last analysis — so callers can always treat
// an empty return as "no context available, proceed normally".
func (s *Scheduler) buildPriceDeltaContext(ctx context.Context, holding *model.Holding) string {
	card, err := s.cardRepo.GetLatestByHolding(ctx, holding.HoldingID)
	if err != nil {
		s.logger.Warn("pre-window: failed to get latest card for price delta",
			zap.Int64("holding_id", holding.HoldingID),
			zap.Error(err),
		)
		return ""
	}
	if card == nil {
		// No prior analysis — nothing to diff against.
		return ""
	}

	lastAnalyzedAt := card.AnalyzedAt

	// Fetch 5 days of price history to cover weekends and short sessions.
	prices, err := s.fetcher().FetchAssetData(ctx, holding.AssetCode, holding.AssetType)
	if err != nil {
		s.logger.Warn("pre-window: failed to fetch price data for delta context",
			zap.String("asset", holding.AssetCode),
			zap.Error(err),
		)
		return ""
	}

	// Collect price bars that fall strictly after lastAnalyzedAt.
	var recent []datasource.PriceData
	for _, p := range prices.Prices {
		if p.Date.After(lastAnalyzedAt) {
			recent = append(recent, p)
		}
	}
	if len(recent) == 0 {
		return ""
	}

	// Find the reference close price (last bar before lastAnalyzedAt).
	var refClose float64
	for _, p := range prices.Prices {
		if !p.Date.After(lastAnalyzedAt) {
			refClose = p.Close
		}
	}

	// Summarize the interval: open of first bar, high/low across all bars,
	// close of last bar, and percent change from refClose if available.
	first := recent[0]
	last := recent[len(recent)-1]

	var high, low float64
	high = first.High
	low = first.Low
	for _, p := range recent[1:] {
		if p.High > high {
			high = p.High
		}
		if p.Low < low {
			low = p.Low
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Since last analysis (%s):\n", lastAnalyzedAt.Format("2006-01-02 15:04 MST"))
	fmt.Fprintf(&sb, "  Open: %.4f, High: %.4f, Low: %.4f, Close: %.4f\n",
		first.Open, high, low, last.Close)

	if refClose > 0 {
		pctChange := (last.Close - refClose) / refClose * 100
		direction := "up"
		if pctChange < 0 {
			direction = "down"
		}
		fmt.Fprintf(&sb, "  Change: %s %.2f%% from prior close %.4f\n",
			direction, abs64(pctChange), refClose)
	}

	return sb.String()
}

// abs64 returns the absolute value of a float64.
func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// buildSubject returns a human-readable subject for the given message type.
func buildSubject(messageType string) string {
	switch messageType {
	case "am_brief":
		return "Morning Market Brief"
	case "pm_digest":
		return "Afternoon Market Digest"
	case "us_pre_brief":
		return "US Pre-Market Brief"
	case "us_digest":
		return "US Market Digest"
	default:
		return "Market Update"
	}
}
