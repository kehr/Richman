package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication and user management business logic.
type Service struct {
	userRepo   *repo.UserRepo
	planRepo   *repo.PlanRepo
	inviteRepo *repo.InviteRepo
	cfg        *config.Config
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

// AuthResult holds the result of a successful authentication operation.
type AuthResult struct {
	User  *model.User `json:"user"`
	Token string      `json:"token"`
}

// Register creates a new user account with invite code validation.
func (s *Service) Register(ctx context.Context, email, password, inviteCode string) (*AuthResult, error) {
	// Validate invite code
	ic, err := s.inviteRepo.GetInviteCodeByCode(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("check invite code: %w", err)
	}
	if ic == nil {
		return nil, model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "invalid invite code")
	}
	if ic.UsedCount >= ic.MaxUses {
		return nil, model.NewAppError(http.StatusBadRequest, "VALIDATION_ERROR", "invite code has reached maximum uses")
	}

	// Check email uniqueness
	existing, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, model.NewAppError(http.StatusConflict, "CONFLICT", "email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Get default plan
	plan, err := s.planRepo.GetPlanByName(ctx, "invite")
	if err != nil {
		return nil, fmt.Errorf("get default plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("default plan 'invite' not found")
	}

	// Create user
	user, err := s.userRepo.CreateUser(ctx, email, string(hash), "user", plan.PlanID)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Increment invite code usage
	if err := s.inviteRepo.IncrementInviteCodeUsage(ctx, ic.InviteCodeID); err != nil {
		return nil, fmt.Errorf("increment invite usage: %w", err)
	}

	// Generate JWT
	token, err := s.GenerateJWT(user.UserID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{User: user, Token: token}, nil
}

// Login authenticates a user with email and password.
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

	token, err := s.GenerateJWT(user.UserID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{User: user, Token: token}, nil
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
