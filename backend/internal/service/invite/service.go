package invite

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"go.uber.org/zap"
)

const (
	codePrefix      = "RM"
	codeRandBytes   = 4 // 4 bytes = 8 hex chars
	maxCodesPerUser = 20
	minUnusedCodes  = 3
	codeRetryMax    = 3
	rewardType      = "extra_analysis_refresh"
)

// bruteRecord tracks failed invite-code validation attempts from a single IP.
type bruteRecord struct {
	failures    int
	windowEnd   time.Time
	lockedUntil time.Time
}

// Service handles invite code generation, consumption, and reward issuance.
type Service struct {
	userInviteCodeRepo *repo.UserInviteCodeRepo
	inviteRewardRepo   *repo.InviteRewardRepo
	userRepo           *repo.UserRepo
	pool               *pgxpool.Pool
	logger             *zap.Logger

	// brute-force protection state (in-memory, per-IP)
	bruteMu sync.Mutex
	brute   map[string]*bruteRecord
}

// NewService creates a new invite Service.
func NewService(
	userInviteCodeRepo *repo.UserInviteCodeRepo,
	inviteRewardRepo *repo.InviteRewardRepo,
	userRepo *repo.UserRepo,
	pool *pgxpool.Pool,
	logger *zap.Logger,
) *Service {
	return &Service{
		userInviteCodeRepo: userInviteCodeRepo,
		inviteRewardRepo:   inviteRewardRepo,
		userRepo:           userRepo,
		pool:               pool,
		logger:             logger,
		brute:              make(map[string]*bruteRecord),
	}
}

// MyCodesResponse is returned by GetMyCodes.
type MyCodesResponse struct {
	Codes        []model.UserInviteCode `json:"codes"`
	TotalCodes   int                    `json:"totalCodes"`
	UsedCount    int                    `json:"usedCount"`
	NextUnlockIn int                    `json:"nextUnlockIn"` // days until next code unlock (7 - streak%7)
}

// MyInvitesResponse is returned by GetMyInvites. Field names match the
// frontend contract (see frontend/src/features/invite/types.ts): the list of
// users appears under "invites" and the total count is precomputed so the UI
// can render the "Invited Users ({n})" header without a separate derivation.
type MyInvitesResponse struct {
	Invites      []model.InvitedUser `json:"invites"`
	TotalInvited int                 `json:"totalInvited"`
}

// generateCode produces a random "RM" + 8-char uppercase hex invite code.
func generateCode() (string, error) {
	b := make([]byte, codeRandBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return codePrefix + hex.EncodeToString(b), nil
}

// CheckBruteForce checks whether the given IP is currently locked out due to
// too many failed invite-code validation attempts. Returns an AppError (429)
// when locked, nil otherwise.
func (s *Service) CheckBruteForce(ip string) error {
	s.bruteMu.Lock()
	defer s.bruteMu.Unlock()

	rec, ok := s.brute[ip]
	if !ok {
		return nil
	}
	now := time.Now()
	if now.Before(rec.lockedUntil) {
		return model.NewAppError(http.StatusTooManyRequests, "RATE_LIMITED",
			"too many failed attempts, please try again later")
	}
	return nil
}

// RecordFailedAttempt increments the failure counter for the given IP.
// After 5 failures within a 10-minute window, the IP is locked for 30 minutes.
func (s *Service) RecordFailedAttempt(ip string) {
	s.bruteMu.Lock()
	defer s.bruteMu.Unlock()

	now := time.Now()
	rec, ok := s.brute[ip]
	if !ok || now.After(rec.windowEnd) {
		// start a new window
		s.brute[ip] = &bruteRecord{
			failures:  1,
			windowEnd: now.Add(10 * time.Minute),
		}
		return
	}
	rec.failures++
	if rec.failures >= 5 {
		rec.lockedUntil = now.Add(30 * time.Minute)
		s.logger.Warn("invite code brute-force lockout triggered",
			zap.String("ip", ip),
			zap.Time("locked_until", rec.lockedUntil),
		)
	}
}

// GenerateCodesForUser generates up to count new invite codes for userID.
// Codes are only generated when unused < 3 AND total < 20. Each code is
// retried up to 3 times on unique-constraint violation.
func (s *Service) GenerateCodesForUser(ctx context.Context, userID int64, count int) error {
	total, err := s.userInviteCodeRepo.CountByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("count user invite codes: %w", err)
	}
	unused, err := s.userInviteCodeRepo.CountUnusedByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("count unused invite codes: %w", err)
	}

	// Determine how many we can actually generate.
	canGenerate := 0
	if unused < minUnusedCodes && total < maxCodesPerUser {
		remaining := maxCodesPerUser - total
		if count < remaining {
			canGenerate = count
		} else {
			canGenerate = remaining
		}
	}

	for i := 0; i < canGenerate; i++ {
		if err := s.createOneCode(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// createOneCode generates and persists a single invite code for userID,
// retrying up to codeRetryMax times on duplicate-key conflicts.
func (s *Service) createOneCode(ctx context.Context, userID int64) error {
	creator := fmt.Sprintf("user:%d", userID)
	for attempt := 0; attempt < codeRetryMax; attempt++ {
		code, err := generateCode()
		if err != nil {
			return err
		}
		_, err = s.userInviteCodeRepo.Create(ctx, userID, code, creator)
		if err == nil {
			return nil
		}
		// Treat duplicate-key errors as a collision and retry.
		// pgx wraps pg error code 23505 (unique_violation) in the error message.
		s.logger.Debug("invite code collision, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}
	return fmt.Errorf("failed to generate unique invite code after %d attempts", codeRetryMax)
}

// createOneCodeWithTx generates and persists a single invite code inside the
// given transaction. Used during registration to keep the entire flow atomic.
func (s *Service) createOneCodeWithTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	creator := fmt.Sprintf("user:%d", userID)
	for attempt := 0; attempt < codeRetryMax; attempt++ {
		code, err := generateCode()
		if err != nil {
			return err
		}
		_, err = s.userInviteCodeRepo.CreateWithTx(ctx, tx, userID, code, creator)
		if err == nil {
			return nil
		}
		s.logger.Debug("invite code collision in tx, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}
	return fmt.Errorf("failed to generate unique invite code (tx) after %d attempts", codeRetryMax)
}

// GenerateInitialCodesWithTx generates initialCount invite codes for a
// newly-registered user inside an existing transaction. Called during the
// registration atomic block to ensure code generation is rolled back if any
// later step fails.
func (s *Service) GenerateInitialCodesWithTx(ctx context.Context, tx pgx.Tx, userID int64, count int) error {
	for i := 0; i < count; i++ {
		if err := s.createOneCodeWithTx(ctx, tx, userID); err != nil {
			return err
		}
	}
	return nil
}

// UseInviteCode atomically marks the invite code as used by newUserID.
// Returns a 409 AppError when the code has already been consumed.
func (s *Service) UseInviteCode(ctx context.Context, tx pgx.Tx, inviteCodeID, newUserID int64) error {
	result, err := s.userInviteCodeRepo.ConsumeCode(ctx, tx, inviteCodeID, newUserID)
	if err != nil {
		return fmt.Errorf("consume invite code: %w", err)
	}
	if result == nil {
		return model.NewAppError(http.StatusConflict, "CONFLICT", "invite code already used")
	}
	return nil
}

// GrantBilateralRewards inserts two reward records inside the given
// transaction: one for the inviter and one for the invitee.
func (s *Service) GrantBilateralRewards(
	ctx context.Context, tx pgx.Tx, inviterID, inviteeID, inviteCodeID int64,
) error {
	creator := fmt.Sprintf("system:invite:%d", inviteCodeID)

	// Reward for inviter.
	if _, err := s.inviteRewardRepo.CreateWithTx(
		ctx, tx, inviterID, rewardType, nil, inviteCodeID, creator,
	); err != nil {
		return fmt.Errorf("grant inviter reward: %w", err)
	}

	// Reward for invitee.
	if _, err := s.inviteRewardRepo.CreateWithTx(
		ctx, tx, inviteeID, rewardType, nil, inviteCodeID, creator,
	); err != nil {
		return fmt.Errorf("grant invitee reward: %w", err)
	}

	return nil
}

// GetMyCodes returns all invite codes for userID along with summary stats.
// nextUnlockIn is the number of days until the next streak-based unlock
// (7 - loginStreak % 7), clamped so a fresh milestone shows 7 (not 0).
func (s *Service) GetMyCodes(ctx context.Context, userID int64) (*MyCodesResponse, error) {
	codes, err := s.userInviteCodeRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user invite codes: %w", err)
	}

	total := len(codes)
	usedCount := 0
	for _, c := range codes {
		if c.IsUsed {
			usedCount++
		}
	}

	// Compute nextUnlockIn from login_streak.
	streak, err := s.userRepo.GetLoginStreak(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get login streak: %w", err)
	}
	nextUnlock := 7 - (streak % 7)
	if nextUnlock == 0 {
		nextUnlock = 7
	}

	if codes == nil {
		codes = []model.UserInviteCode{}
	}

	return &MyCodesResponse{
		Codes:        codes,
		TotalCodes:   total,
		UsedCount:    usedCount,
		NextUnlockIn: nextUnlock,
	}, nil
}

// GetMyInvites returns info about users who were invited by userID.
func (s *Service) GetMyInvites(ctx context.Context, userID int64) (*MyInvitesResponse, error) {
	users, err := s.userInviteCodeRepo.ListInvitedUsers(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list invited users: %w", err)
	}
	if users == nil {
		users = []model.InvitedUser{}
	}
	return &MyInvitesResponse{Invites: users, TotalInvited: len(users)}, nil
}

// GetFirstAvailableCode returns the first unused invite code for userID, or
// an empty string when none exist. Used by the share-link endpoint.
func (s *Service) GetFirstAvailableCode(ctx context.Context, userID int64) (string, error) {
	code, err := s.userInviteCodeRepo.GetFirstAvailable(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get first available invite code: %w", err)
	}
	if code == nil {
		return "", nil
	}
	return code.Code, nil
}

// MaybeGenerateStreakCode checks whether a login-streak milestone was reached
// and generates one extra invite code when the conditions are met:
//   - streak % 7 == 0 (milestone)
//   - unused < 3
//   - total < 20
func (s *Service) MaybeGenerateStreakCode(ctx context.Context, userID int64, streak int) error {
	if streak == 0 || streak%7 != 0 {
		return nil
	}
	return s.GenerateCodesForUser(ctx, userID, 1)
}

// LookupPersonalCode finds a personal invite code by its code string.
// Returns nil when no active (is_deleted = 0) code with that value exists.
// The caller must check IsUsed before proceeding with consumption.
func (s *Service) LookupPersonalCode(ctx context.Context, code string) (*model.UserInviteCode, error) {
	return s.userInviteCodeRepo.GetByCode(ctx, code)
}

// GetLoginStreak is a thin pass-through so callers that hold only an
// *invite.Service do not need to import repo directly.
func (s *Service) GetLoginStreak(ctx context.Context, userID int64) (int, error) {
	return s.userRepo.GetLoginStreak(ctx, userID)
}

// Pool exposes the underlying pgxpool so auth.Service can open transactions
// for the registration flow without importing pgxpool directly.
func (s *Service) Pool() *pgxpool.Pool {
	return s.pool
}

// ClearUsedByForUser nullifies used_by_user_id on all invite codes that were
// consumed by the given user. Called during account deletion so the invite codes
// are no longer linked to the soft-deleted user record. Thin pass-through to
// the underlying repo method.
func (s *Service) ClearUsedByForUser(ctx context.Context, userID int64) error {
	return s.userInviteCodeRepo.ClearUsedByForUser(ctx, userID)
}
