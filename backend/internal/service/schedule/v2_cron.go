package schedule

import (
	"context"
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/richson"
	"github.com/richman/backend/internal/service/emailpush"
)

// HoldingAnalyzer is the subset of analysis.V2HoldingAnalyzer used by the v2
// cron tasks. Defined as a local interface to break the circular import between
// the schedule and analysis packages (analysis/scheduler.go imports schedule).
type HoldingAnalyzer interface {
	AnalyzeHolding(ctx context.Context, userID, holdingID int64) (*model.DecisionCard, error)
}

// reAShareCode matches six-digit numeric codes that identify A-share stocks.
var reAShareCode = regexp.MustCompile(`^\d{6}$`)

// holdingAnalysisConcurrency is the maximum number of users processed in
// parallel during the daily holding analysis cron task.
const holdingAnalysisConcurrency = 5

// dailyAlertedTracker records which asset codes have already fired a market
// alert in the current UTC day. It backs the unidirectional dedup between
// the 06:00 score change alert and the 15:30 A-share closing alert
// (richman TRD SS22.14).
//
// A database-backed dedup is not possible without a schema change:
// rm_notification_logs is a per-user log and carries no asset/event column.
// An in-memory set is acceptable for MVP because the scheduler runs inside
// a single process and A-share assets are not yet enabled in production.
// First write of each new UTC day wipes the previous day's entries.
var dailyAlertedTracker = struct {
	mu  sync.Mutex
	day time.Time // UTC day boundary (00:00:00 UTC) the set belongs to
	set map[string]struct{}
}{
	set: make(map[string]struct{}),
}

// currentUTCDay returns today's date in UTC truncated to the day start.
func currentUTCDay() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// dailyAlertedToday reports whether markDailyAlerted was called with code
// earlier in the current UTC day.
func dailyAlertedToday(code string) bool {
	dailyAlertedTracker.mu.Lock()
	defer dailyAlertedTracker.mu.Unlock()
	today := currentUTCDay()
	if !dailyAlertedTracker.day.Equal(today) {
		return false
	}
	_, ok := dailyAlertedTracker.set[code]
	return ok
}

// markDailyAlerted records that a market alert was emitted for this asset
// in the current UTC day. The first call on a new UTC day wipes the prior
// day's entries before recording.
func markDailyAlerted(code string) {
	dailyAlertedTracker.mu.Lock()
	defer dailyAlertedTracker.mu.Unlock()
	today := currentUTCDay()
	if !dailyAlertedTracker.day.Equal(today) {
		dailyAlertedTracker.day = today
		dailyAlertedTracker.set = make(map[string]struct{})
	}
	dailyAlertedTracker.set[code] = struct{}{}
}

// RegisterV2CronJobs adds all v2 scheduled tasks to the given cron instance.
// The caller is responsible for starting and stopping c; this function only
// registers entries.
func RegisterV2CronJobs(
	c *cron.Cron,
	richsonClient *richson.Client,
	emailPushSvc *emailpush.Service,
	holdingAnalyzer HoldingAnalyzer,
	eventAlertRepo *repo.EventAlertReadRepo,
	analysisJobRepo *repo.AnalysisJobReadRepo,
	assetRepo *repo.AssetRepo,
	analysisReadRepo *repo.AssetAnalysisReadRepo,
	holdingRepo *repo.HoldingRepo,
	userRepo *repo.UserRepo,
	notifLogRepo *repo.NotificationLogRepo,
	cfg *config.Config,
	logger *zap.Logger,
) {
	// Per-task mutexes prevent overlapping runs of long-running tasks.
	var (
		assetAnalysisMu   sync.Mutex
		holdingAnalysisMu sync.Mutex
		briefingMu        sync.Mutex
	)

	// Build the platform LLM config once from application config.
	platformLLM := buildPlatformLLMConfig(cfg)

	// addCronFunc registers a cron entry and logs a fatal error if registration
	// fails (invalid spec). All specs are constants so a failure is a programming
	// error that must be surfaced immediately rather than silently skipped.
	addCronFunc := func(spec string, fn func()) {
		if _, err := c.AddFunc(spec, fn); err != nil {
			logger.Error("failed to register v2 cron entry",
				zap.String("spec", spec),
				zap.Error(err),
			)
		}
	}

	// ---- 1. Daily asset analysis: 22:00 UTC (06:00 UTC+8) ----
	addCronFunc("0 22 * * *", func() {
		if !assetAnalysisMu.TryLock() {
			logger.Warn("daily asset analysis already running, skipping this round")
			return
		}
		defer assetAnalysisMu.Unlock()
		runDailyAssetAnalysis(richsonClient, emailPushSvc, assetRepo, analysisReadRepo, notifLogRepo, platformLLM, logger)
	})

	// ---- 2. Daily holding analysis: 23:30 UTC (07:30 UTC+8) ----
	addCronFunc("30 23 * * *", func() {
		if !holdingAnalysisMu.TryLock() {
			logger.Warn("daily holding analysis already running, skipping this round")
			return
		}
		defer holdingAnalysisMu.Unlock()
		runDailyHoldingAnalysis(holdingAnalyzer, emailPushSvc, holdingRepo, logger)
	})

	// ---- 3. Daily briefing email: 00:30 UTC (08:30 UTC+8) ----
	addCronFunc("30 0 * * *", func() {
		if !briefingMu.TryLock() {
			logger.Warn("daily briefing already running, skipping this round")
			return
		}
		defer briefingMu.Unlock()
		runDailyBriefing(emailPushSvc, analysisReadRepo, logger)
	})

	// ---- 4. A-share closing alert: 07:30 UTC (15:30 UTC+8) Monday–Friday ----
	addCronFunc("30 7 * * 1-5", func() {
		runAShareClosingAlert(emailPushSvc, assetRepo, analysisReadRepo, notifLogRepo, logger)
	})

	// ---- 5. Weekly insight email: 00:30 UTC Monday (08:30 UTC+8 Monday) ----
	addCronFunc("30 0 * * 1", func() {
		runWeeklyInsight(emailPushSvc, logger)
	})

	// ---- 6. Event alert polling: top of every hour ----
	addCronFunc("0 * * * *", func() {
		runEventAlertPolling(emailPushSvc, eventAlertRepo, logger)
	})

	// ---- 7. Expired job cleanup: every 3 minutes ----
	// Interval tightened from 10 -> 3 minutes (richman TRD SS22.4). When a
	// richson worker crashes mid-job the row sits in `running` status under a
	// unique (asset_code) constraint, so any retry blocks until this sweep
	// clears it. 3 minutes keeps the worst-case block time bounded without
	// flooding Postgres with empty scans.
	addCronFunc("*/3 * * * *", func() {
		runExpiredJobCleanup(analysisJobRepo, logger)
	})

	// ---- 8. richson health check: every 30 seconds ----
	addCronFunc("@every 30s", func() {
		runRichsonHealthCheck(richsonClient, logger)
	})

	logger.Info("v2 cron jobs registered",
		zap.Int("count", 8),
	)
}

// ---- buildPlatformLLMConfig ----

// buildPlatformLLMConfig constructs a richson.LLMConfig from the application
// config's LLM section. The provider key/model selection follows the same
// priority used by the LLM resolver: claude first, then openai.
func buildPlatformLLMConfig(cfg *config.Config) *richson.LLMConfig {
	switch cfg.LLM.Provider {
	case "openai":
		if cfg.LLM.OpenAIAPIKey != "" {
			return &richson.LLMConfig{
				Provider: "openai",
				Model:    cfg.LLM.OpenAIModel,
				APIKey:   cfg.LLM.OpenAIAPIKey,
			}
		}
	default:
		if cfg.LLM.ClaudeAPIKey != "" {
			return &richson.LLMConfig{
				Provider: "claude",
				Model:    cfg.LLM.ClaudeModel,
				APIKey:   cfg.LLM.ClaudeAPIKey,
			}
		}
	}
	return nil
}

// ---- Task 1: Daily asset analysis ----

func runDailyAssetAnalysis(
	richsonClient *richson.Client,
	emailPushSvc *emailpush.Service,
	assetRepo *repo.AssetRepo,
	analysisReadRepo *repo.AssetAnalysisReadRepo,
	notifLogRepo *repo.NotificationLogRepo,
	llmCfg *richson.LLMConfig,
	logger *zap.Logger,
) {
	ctx := context.Background()
	logger.Info("daily asset analysis started")

	assets, err := assetRepo.ListActiveWithType(ctx, "")
	if err != nil {
		logger.Error("daily asset analysis: list assets failed", zap.Error(err))
		return
	}
	if len(assets) == 0 {
		logger.Info("daily asset analysis: no active assets")
		return
	}

	batchAssets := make([]richson.BatchAnalyzeAsset, 0, len(assets))
	codes := make([]string, 0, len(assets))
	for _, a := range assets {
		batchAssets = append(batchAssets, richson.BatchAnalyzeAsset{
			AssetCode: a.Code,
			Locale:    "zh",
		})
		codes = append(codes, a.Code)
	}

	req := richson.TriggerBatchAnalysisRequest{
		Assets:    batchAssets,
		LLMConfig: llmCfg,
	}

	resp, err := richsonClient.TriggerBatchAnalysis(ctx, req)
	if err != nil {
		logger.Error("daily asset analysis: trigger batch failed", zap.Error(err))
		return
	}

	logger.Info("daily asset analysis triggered",
		zap.Int("jobs", len(resp.Jobs)),
		zap.Int("skipped", len(resp.Skipped)),
	)

	// After batch returns, check for significant score changes (score change alert).
	// Run inline after triggering since TriggerBatchAnalysis is async and jobs may
	// not be complete yet; score change alert checks the latest persisted analyses.
	runScoreChangeAlert(emailPushSvc, analysisReadRepo, codes, logger)
}

// ---- Task: Score change alert (embedded in daily analysis flow) ----

func runScoreChangeAlert(
	emailPushSvc *emailpush.Service,
	analysisReadRepo *repo.AssetAnalysisReadRepo,
	codes []string,
	logger *zap.Logger,
) {
	ctx := context.Background()
	logger.Info("score change alert check started")

	latestMap, err := analysisReadRepo.GetLatestByAssetCodes(ctx, codes)
	if err != nil {
		logger.Error("score change alert: get latest analyses failed", zap.Error(err))
		return
	}

	// Build-and-send is collapsed into one loop so each successful push can
	// call markDailyAlerted(a.AssetCode) — feeding the unidirectional dedup
	// consumed by the 15:30 A-share closing alert (richman TRD SS22.14).
	var sent, eligible int
	for _, a := range latestMap {
		if a.ScoreDelta == nil {
			continue
		}
		if math.Abs(*a.ScoreDelta) < 10 {
			continue
		}
		eligible++
		dir := scoreChangeDirection(*a.ScoreDelta)
		alert := &model.EventAlert{
			EventSlug:       "score_change_" + a.AssetCode,
			EventTitle:      a.AssetCode + " score change: " + formatScoreDelta(*a.ScoreDelta),
			PrevProbability: a.OverallScore - *a.ScoreDelta,
			CurrProbability: a.OverallScore,
			Delta:           *a.ScoreDelta,
			Threshold:       10,
			GoldDirection:   &dir,
		}
		if err := emailPushSvc.SendMarketAlert(ctx, alert); err != nil {
			logger.Error("score change alert: send failed",
				zap.String("event_slug", alert.EventSlug),
				zap.Error(err),
			)
			continue
		}
		markDailyAlerted(a.AssetCode)
		sent++
	}

	if eligible == 0 {
		logger.Info("score change alert: no significant changes found")
		return
	}
	logger.Info("score change alert: sent alerts",
		zap.Int("sent", sent),
		zap.Int("eligible", eligible),
	)
}

// scoreChangeDirection maps a score delta to a directional label.
func scoreChangeDirection(delta float64) string {
	if delta > 0 {
		return "bullish"
	}
	return "bearish"
}

// formatScoreDelta formats a score delta as a signed string (e.g. "+12.5").
func formatScoreDelta(delta float64) string {
	if delta >= 0 {
		return "+" + formatFloat(delta)
	}
	return formatFloat(delta)
}

func formatFloat(f float64) string {
	// Use strconv-free formatting: round to one decimal place.
	abs := f
	if abs < 0 {
		abs = -abs
	}
	intPart := int(abs)
	fracPart := int((abs-float64(intPart))*10 + 0.5)
	if fracPart >= 10 {
		intPart++
		fracPart = 0
	}
	sign := ""
	if f < 0 {
		sign = "-"
	}
	return sign + itoa(intPart) + "." + itoa(fracPart)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// ---- Task 2: Daily holding analysis ----

func runDailyHoldingAnalysis(
	holdingAnalyzer HoldingAnalyzer,
	emailPushSvc *emailpush.Service,
	holdingRepo *repo.HoldingRepo,
	logger *zap.Logger,
) {
	ctx := context.Background()
	logger.Info("daily holding analysis started")

	userIDs, err := holdingRepo.ListUsersWithHoldings(ctx)
	if err != nil {
		logger.Error("daily holding analysis: list users failed", zap.Error(err))
		return
	}
	if len(userIDs) == 0 {
		logger.Info("daily holding analysis: no users with holdings")
		return
	}

	sem := make(chan struct{}, holdingAnalysisConcurrency)
	var wg sync.WaitGroup

	for _, uid := range userIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(userID int64) {
			defer wg.Done()
			defer func() { <-sem }()
			analyzeUserHoldings(ctx, userID, holdingAnalyzer, emailPushSvc, holdingRepo, logger)
		}(uid)
	}
	wg.Wait()

	logger.Info("daily holding analysis completed", zap.Int("users", len(userIDs)))
}

// analyzeUserHoldings runs holding analysis for all active holdings of one
// user, serially within the user scope to avoid conflicting writes.
func analyzeUserHoldings(
	ctx context.Context,
	userID int64,
	holdingAnalyzer HoldingAnalyzer,
	emailPushSvc *emailpush.Service,
	holdingRepo *repo.HoldingRepo,
	logger *zap.Logger,
) {
	holdings, err := holdingRepo.ListHoldingsByUser(ctx, userID)
	if err != nil {
		logger.Error("holding analysis: list holdings failed",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return
	}

	for _, h := range holdings {
		card, err := holdingAnalyzer.AnalyzeHolding(ctx, userID, h.HoldingID)
		if err != nil {
			logger.Error("holding analysis: analyze failed",
				zap.Int64("user_id", userID),
				zap.Int64("holding_id", h.HoldingID),
				zap.Error(err),
			)
			continue
		}

		// Send holding suggestion when action advice is available.
		if card.ActionAdvice != "" {
			if err := emailPushSvc.SendHoldingSuggestion(ctx, userID, card); err != nil {
				logger.Error("holding analysis: send suggestion failed",
					zap.Int64("user_id", userID),
					zap.Int64("holding_id", h.HoldingID),
					zap.Error(err),
				)
			}
		}
	}
}

// ---- Task 3: Daily briefing ----

func runDailyBriefing(
	emailPushSvc *emailpush.Service,
	analysisReadRepo *repo.AssetAnalysisReadRepo,
	logger *zap.Logger,
) {
	ctx := context.Background()
	logger.Info("daily briefing started")

	// Check data freshness: latest gold analysis should be after today 22:00 UTC
	// (06:00 UTC+8). Log a warning if stale but continue with available data.
	goldAnalysis, err := analysisReadRepo.GetLatestByAssetCode(ctx, "GLD")
	if err != nil {
		logger.Warn("daily briefing: failed to check gold analysis freshness", zap.Error(err))
	} else if goldAnalysis != nil {
		threshold := todayUTC22h00()
		if goldAnalysis.AnalyzedAt.Before(threshold) {
			logger.Warn("daily briefing: gold analysis is stale",
				zap.Time("analyzed_at", goldAnalysis.AnalyzedAt),
				zap.Time("expected_after", threshold),
			)
		}
	}

	if err := emailPushSvc.SendDailyBriefing(ctx); err != nil {
		logger.Error("daily briefing: send failed", zap.Error(err))
		return
	}
	logger.Info("daily briefing completed")
}

// todayUTC22h00 returns the 22:00 UTC timestamp for the current UTC date.
// This corresponds to 06:00 UTC+8, after which the daily asset analysis should
// have run.
func todayUTC22h00() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, time.UTC)
}

// ---- Task 4: A-share closing alert ----

func runAShareClosingAlert(
	emailPushSvc *emailpush.Service,
	assetRepo *repo.AssetRepo,
	analysisReadRepo *repo.AssetAnalysisReadRepo,
	notifLogRepo *repo.NotificationLogRepo,
	logger *zap.Logger,
) {
	ctx := context.Background()
	logger.Info("A-share closing alert check started")

	// Query all A-share assets (6-digit numeric code).
	allAssets, err := assetRepo.ListActiveWithType(ctx, "")
	if err != nil {
		logger.Error("A-share closing alert: list assets failed", zap.Error(err))
		return
	}

	var aShareCodes []string
	for _, a := range allAssets {
		if reAShareCode.MatchString(a.Code) {
			aShareCodes = append(aShareCodes, a.Code)
		}
	}
	if len(aShareCodes) == 0 {
		logger.Info("A-share closing alert: no A-share assets found")
		return
	}

	latestMap, err := analysisReadRepo.GetLatestByAssetCodes(ctx, aShareCodes)
	if err != nil {
		logger.Error("A-share closing alert: get latest analyses failed", zap.Error(err))
		return
	}

	// Build-and-send is collapsed into one loop so the dedup check and the
	// markDailyAlerted call share the per-asset context (richman TRD SS22.14).
	//
	// Dedup is unidirectional: 15:30 closing alert skips assets the 06:00
	// analysis already pushed. Reverse dedup (06:00 skipping assets that
	// 15:30 touched) is not needed because 06:00 always runs first in the
	// same UTC day.
	//
	// Implementation trade-off: rm_notification_logs is a per-user log with
	// no asset/event column, so a pure SQL per-asset lookup is impossible
	// without a schema change. For MVP we keep an in-memory set of asset
	// codes that fired earlier today (see dailyAlertedTracker). When A-share
	// assets are re-enabled in Phase 2 the set grows from the 06:00
	// runScoreChangeAlert path and is consulted here.
	var sent, eligible int
	for _, a := range latestMap {
		if a.ScoreDelta == nil {
			continue
		}
		if math.Abs(*a.ScoreDelta) < 5 {
			continue
		}

		if dailyAlertedToday(a.AssetCode) {
			logger.Debug("A-share closing alert: already alerted today, skipping",
				zap.String("asset_code", a.AssetCode),
			)
			continue
		}

		eligible++
		dir := scoreChangeDirection(*a.ScoreDelta)
		alert := &model.EventAlert{
			EventSlug:       "ashare_close_" + a.AssetCode,
			EventTitle:      a.AssetCode + " A-share closing: " + formatScoreDelta(*a.ScoreDelta),
			PrevProbability: a.OverallScore - *a.ScoreDelta,
			CurrProbability: a.OverallScore,
			Delta:           *a.ScoreDelta,
			Threshold:       5,
			GoldDirection:   &dir,
		}
		if err := emailPushSvc.SendMarketAlert(ctx, alert); err != nil {
			logger.Error("A-share closing alert: send failed",
				zap.String("event_slug", alert.EventSlug),
				zap.Error(err),
			)
			continue
		}
		markDailyAlerted(a.AssetCode)
		sent++
	}

	if eligible == 0 {
		logger.Info("A-share closing alert: no changes meeting threshold")
		return
	}
	logger.Info("A-share closing alert: sent alerts",
		zap.Int("sent", sent),
		zap.Int("eligible", eligible),
	)
}

// ---- Task 5: Weekly insight ----

func runWeeklyInsight(
	emailPushSvc *emailpush.Service,
	logger *zap.Logger,
) {
	ctx := context.Background()
	logger.Info("weekly insight started")

	if err := emailPushSvc.SendWeeklyInsight(ctx); err != nil {
		logger.Error("weekly insight: send failed", zap.Error(err))
		return
	}
	logger.Info("weekly insight completed")
}

// ---- Task 6: Event alert polling ----

func runEventAlertPolling(
	emailPushSvc *emailpush.Service,
	eventAlertRepo *repo.EventAlertReadRepo,
	logger *zap.Logger,
) {
	ctx := context.Background()

	alerts, err := eventAlertRepo.GetUnalerted(ctx)
	if err != nil {
		logger.Error("event alert polling: get unalerted failed", zap.Error(err))
		return
	}
	if len(alerts) == 0 {
		return
	}

	logger.Info("event alert polling: processing alerts", zap.Int("count", len(alerts)))

	var processedIDs []int64
	for _, a := range alerts {
		alertCopy := a // avoid loop variable capture
		if err := emailPushSvc.SendMarketAlert(ctx, &alertCopy); err != nil {
			logger.Error("event alert polling: send failed",
				zap.String("event_slug", a.EventSlug),
				zap.Error(err),
			)
			// Do not mark as alerted on send failure; will retry next poll.
			continue
		}
		processedIDs = append(processedIDs, a.ID)
	}

	if len(processedIDs) == 0 {
		return
	}

	if err := eventAlertRepo.MarkAlerted(ctx, processedIDs); err != nil {
		logger.Error("event alert polling: mark alerted failed",
			zap.Int("count", len(processedIDs)),
			zap.Error(err),
		)
	} else {
		logger.Info("event alert polling: marked alerted",
			zap.Int("count", len(processedIDs)),
		)
	}
}

// ---- Task 7: Expired job cleanup ----

func runExpiredJobCleanup(
	analysisJobRepo *repo.AnalysisJobReadRepo,
	logger *zap.Logger,
) {
	ctx := context.Background()

	n, err := analysisJobRepo.FailExpiredJobs(ctx)
	if err != nil {
		logger.Error("expired job cleanup: fail expired jobs failed", zap.Error(err))
		return
	}
	if n > 0 {
		logger.Info("expired job cleanup: failed expired jobs", zap.Int64("count", n))
	}
}

// ---- Task 8: richson health check ----

func runRichsonHealthCheck(
	richsonClient *richson.Client,
	logger *zap.Logger,
) {
	ctx := context.Background()
	if _, err := richsonClient.HealthCheck(ctx); err != nil {
		// HealthCheck already logs a Warn internally; add context here.
		logger.Debug("richson health check: unhealthy", zap.Error(err))
	}
}
