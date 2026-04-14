package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	inviteSvc "github.com/richman/backend/internal/service/invite"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication and user management business logic.
type Service struct {
	userRepo      *repo.UserRepo
	planRepo      *repo.PlanRepo
	inviteRepo    *repo.InviteRepo
	inviteService *inviteSvc.Service
	cfg           *config.Config
}

// NewService creates a new auth Service.
func NewService(
	userRepo *repo.UserRepo,
	planRepo *repo.PlanRepo,
	inviteRepo *repo.InviteRepo,
	cfg *config.Config,
) *Service {
	return &Service{
		userRepo:   userRepo,
		planRepo:   planRepo,
		inviteRepo: inviteRepo,
		cfg:        cfg,
	}
}

// NewServiceWithInvite creates a new auth Service with invite integration.
// Use this constructor when the invite system is fully wired.
func NewServiceWithInvite(
	userRepo *repo.UserRepo,
	planRepo *repo.PlanRepo,
	inviteRepo *repo.InviteRepo,
	invite *inviteSvc.Service,
	cfg *config.Config,
) *Service {
	return &Service{
		userRepo:      userRepo,
		planRepo:      planRepo,
		inviteRepo:    inviteRepo,
		inviteService: invite,
		cfg:           cfg,
	}
}

// AuthResult holds the result of a successful authentication operation.
type AuthResult struct {
	User  *model.User `json:"user"`
	Token string      `json:"token"`
}

// ValidatePasswordComplexity enforces the policy from richman TRD SS22.5:
// at least one uppercase letter, one lowercase letter, and one digit. Length
// limits are enforced by binding tags at the handler layer so this helper
// assumes the string has already passed the 8..128 byte check. Exposed so
// future ChangePassword flows can apply the same rule.
func ValidatePasswordComplexity(password string) error {
	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
		if hasUpper && hasLower && hasDigit {
			return nil
		}
	}
	return model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR",
		"password must contain uppercase, lowercase, and digit characters")
}

// Register creates a new user account with invite code validation.
// disclaimerAccepted must be true; a false value returns a 400 error.
// The invite code is checked against global codes first (v1 flow); if not found
// there it falls back to personal user invite codes (v2 flow) and, on success,
// runs the full personal-invite transaction: consume code, grant bilateral
// rewards, and generate 3 initial invite codes for the new user.
func (s *Service) Register(
	ctx context.Context, email, password, inviteCode string, disclaimerAccepted bool,
) (*AuthResult, error) {
	// Validate disclaimer acceptance.
	if !disclaimerAccepted {
		return nil, model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR",
			"disclaimer must be accepted to register")
	}

	// Enforce password character-class complexity (richman TRD SS22.5).
	if err := ValidatePasswordComplexity(password); err != nil {
		return nil, err
	}

	// --- Phase 1: try global (v1) invite codes ---
	ic, err := s.inviteRepo.GetInviteCodeByCode(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("check invite code: %w", err)
	}
	if ic != nil {
		// Found a global code.
		if ic.UsedCount >= ic.MaxUses {
			return nil, model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR",
				"invite code has reached maximum uses")
		}
		return s.registerWithGlobalCode(ctx, email, password, ic)
	}

	// --- Phase 2: try personal (v2) invite codes ---
	if s.inviteService == nil {
		// Invite service not wired; treat missing code as invalid.
		return nil, model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "invalid invite code")
	}

	personalCode, err := s.inviteService.LookupPersonalCode(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("check personal invite code: %w", err)
	}
	if personalCode == nil {
		return nil, model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "invalid invite code")
	}
	if personalCode.IsUsed {
		return nil, model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "invite code already used")
	}

	return s.registerWithPersonalCode(ctx, email, password, personalCode)
}

// registerWithGlobalCode handles the v1 registration path (global invite codes).
func (s *Service) registerWithGlobalCode(
	ctx context.Context, email, password string, ic *model.InviteCode,
) (*AuthResult, error) {
	// Check email uniqueness.
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, model.NewAppError(http.StatusConflict, "CONFLICT", "email already registered")
	}

	// Hash password.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Get default plan.
	plan, err := s.planRepo.GetPlanByName(ctx, "invite")
	if err != nil {
		return nil, fmt.Errorf("get default plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("default plan 'invite' not found")
	}

	// Create user.
	user, err := s.userRepo.CreateUser(ctx, email, string(hash), "user", plan.PlanID)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Record disclaimer acceptance (PRD SS12 / migration 022 column).
	if err := s.userRepo.MarkDisclaimerAccepted(ctx, user.UserID); err != nil {
		return nil, fmt.Errorf("mark disclaimer accepted: %w", err)
	}

	// Increment global invite code usage.
	if err := s.inviteRepo.IncrementInviteCodeUsage(ctx, ic.InviteCodeID); err != nil {
		return nil, fmt.Errorf("increment invite usage: %w", err)
	}

	// Generate JWT.
	token, err := s.GenerateJWT(user.UserID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{User: user, Token: token}, nil
}

// registerWithPersonalCode handles the v2 registration path (personal invite codes).
// The entire flow runs inside a single transaction for atomicity.
func (s *Service) registerWithPersonalCode(
	ctx context.Context, email, password string, personalCode *model.UserInviteCode,
) (*AuthResult, error) {
	// Check email uniqueness before opening the transaction.
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, model.NewAppError(http.StatusConflict, "CONFLICT", "email already registered")
	}

	// Hash password.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Get default plan.
	plan, err := s.planRepo.GetPlanByName(ctx, "invite")
	if err != nil {
		return nil, fmt.Errorf("get default plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("default plan 'invite' not found")
	}

	// Open transaction.
	tx, err := s.inviteService.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin registration tx: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Create user inside transaction.
	user, err := s.userRepo.CreateUserWithTx(ctx, tx, email, string(hash), "user", plan.PlanID)
	if err != nil {
		return nil, fmt.Errorf("create user (tx): %w", err)
	}

	// Record disclaimer acceptance inside the same transaction so account
	// creation and consent stay atomic (PRD SS12 / migration 022).
	if err := s.userRepo.MarkDisclaimerAcceptedWithTx(ctx, tx, user.UserID); err != nil {
		return nil, fmt.Errorf("mark disclaimer accepted (tx): %w", err)
	}

	// Consume personal invite code.
	if err := s.inviteService.UseInviteCode(ctx, tx, personalCode.InviteCodeID, user.UserID); err != nil {
		return nil, err
	}

	// Grant bilateral rewards (inviter + invitee).
	if err := s.inviteService.GrantBilateralRewards(
		ctx, tx, personalCode.UserID, user.UserID, personalCode.InviteCodeID,
	); err != nil {
		return nil, fmt.Errorf("grant bilateral rewards: %w", err)
	}

	// Generate 3 initial invite codes for the new user.
	if err := s.inviteService.GenerateInitialCodesWithTx(ctx, tx, user.UserID, 3); err != nil {
		return nil, fmt.Errorf("generate initial codes: %w", err)
	}

	// Commit.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit registration tx: %w", err)
	}
	tx = nil // prevent deferred rollback

	// Generate JWT.
	token, err := s.GenerateJWT(user.UserID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{User: user, Token: token}, nil
}

// Login authenticates a user with email and password. On success, the login
// streak is updated and a new invite code is generated when a 7-day milestone
// is reached.
func (s *Service) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		}
		return nil, fmt.Errorf("verify password: %w", err)
	}

	// Update login streak. The date boundary is pinned to Asia/Shanghai inside
	// the SQL itself (see UserRepo.UpdateLoginStreak, invite TRD SS11.2) so the
	// result is independent of the server / session timezone configuration.
	streak, streakErr := s.userRepo.UpdateLoginStreak(ctx, user.UserID)
	if streakErr != nil {
		// Non-fatal: a failed streak update should not block login.
		zap.L().Warn("update login streak failed",
			zap.Int64("userID", user.UserID),
			zap.Error(streakErr),
		)
	}

	// On 7-day streak milestone, attempt to grant a new invite code.
	if streak > 0 && s.inviteService != nil {
		if genErr := s.inviteService.MaybeGenerateStreakCode(ctx, user.UserID, streak); genErr != nil {
			// Non-fatal: log and continue.
			zap.L().Warn("generate streak invite code failed",
				zap.Int64("userID", user.UserID),
				zap.Error(genErr),
			)
		}
	}

	token, err := s.GenerateJWT(user.UserID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{User: user, Token: token}, nil
}

// DeleteAccount soft-deletes the authenticated user after verifying the
// provided password. Invite code back-references (used_by_user_id) are
// cleared as a best-effort follow-up; their failure does not block deletion.
func (s *Service) DeleteAccount(ctx context.Context, userID int64, password string) error {
	// Verify password before deletion.
	hash, err := s.userRepo.GetPasswordHash(ctx, userID)
	if err != nil {
		return fmt.Errorf("load password hash: %w", err)
	}
	if hash == "" {
		return model.NewAppError(http.StatusNotFound, "NOT_FOUND", "user not found")
	}
	if bcryptErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); bcryptErr != nil {
		return model.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "incorrect password")
	}

	modifier := fmt.Sprintf("user:%d:self-delete", userID)
	if delErr := s.userRepo.SoftDeleteUser(ctx, userID, modifier); delErr != nil {
		return fmt.Errorf("soft delete user: %w", delErr)
	}

	// Clear invite code back-references; non-fatal on failure.
	if s.inviteService != nil {
		if clearErr := s.inviteService.ClearUsedByForUser(ctx, userID); clearErr != nil {
			zap.L().Warn("clear invite code used_by_user_id failed after account deletion",
				zap.Int64("user_id", userID),
				zap.Error(clearErr),
			)
		}
	}

	return nil
}

// GetCurrentUser retrieves a user by ID.
func (s *Service) GetCurrentUser(ctx context.Context, userID int64) (*model.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, model.ErrNotFound
	}
	return user, nil
}

// Claims represents the JWT claims payload.
type Claims struct {
	UserID int64  `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a signed JWT token for the given user.
//
// MVP (current): single long-lived access token, cfg.JWT.Expiry defaults to
// 7 days. There is no refresh mechanism -- when the token expires the client
// must re-authenticate.
//
// Phase 2 (planned, richman TRD SS22.6): replace this with a two-token flow
// -- a short-lived access token (~15min) plus a long-lived refresh token
// (~30 days) rotated through a dedicated /auth/refresh endpoint, with the
// refresh token persisted (hashed) so it can be revoked at logout.
func (s *Service) GenerateJWT(userID int64, email, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWT.Expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "richman-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ValidateJWT parses and validates a JWT token string.
func (s *Service) ValidateJWT(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(_ *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
