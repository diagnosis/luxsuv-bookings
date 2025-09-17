package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/diagnosis/luxsuv-bookings/pkg/auth"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/events"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/mailer"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AuthService interface {
	Register(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, string, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)
	VerifyEmail(ctx context.Context, token string) (*domain.User, error)
	ResendVerification(ctx context.Context, email string) error
	RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResponse, error)
	GetUser(ctx context.Context, id int64) (*domain.User, error)
	UpdateUser(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error)
	DeleteUser(ctx context.Context, id int64) error
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error)
	UpdateUserRole(ctx context.Context, userID int64, role string) error
}

type authService struct {
	userRepo   repository.UserRepository
	verifyRepo repository.VerifyRepository
	mailer     mailer.Service
	eventBus   events.EventBus
	config     *config.Config
}

func NewAuthService(
	userRepo repository.UserRepository,
	verifyRepo repository.VerifyRepository,
	mailer mailer.Service,
	eventBus events.EventBus,
	config *config.Config,
) AuthService {
	return &authService{
		userRepo:   userRepo,
		verifyRepo: verifyRepo,
		mailer:     mailer,
		eventBus:   eventBus,
		config:     config,
	}
}

func (s *authService) Register(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, string, error) {
	// Normalize and validate
	req.Normalize()
	if err := req.Validate(); err != nil {
		return nil, "", fmt.Errorf("validation failed: %w", err)
	}
	
	// Check if user already exists
	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil && err != pgx.ErrNoRows {
		return nil, "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, "", fmt.Errorf("user with this email already exists")
	}
	
	// Hash password
	passwordHash, err := argon2id.CreateHash(req.Password, argon2id.DefaultParams)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}
	
	// Create user
	user, err := s.userRepo.Create(ctx, req, passwordHash)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}
	
	// Link existing bookings if any
	if err := s.userRepo.LinkExistingBookings(ctx, user.ID, user.Email); err != nil {
		logger.WarnContext(ctx, "Failed to link existing bookings", "error", err, "user_id", user.ID)
	}
	
	// Create verification token
	verifyToken := uuid.NewString()
	expiresAt := time.Now().Add(s.config.Auth.EmailVerificationTTL)
	
	if err := s.verifyRepo.CreateEmailVerification(ctx, user.ID, verifyToken, expiresAt); err != nil {
		logger.ErrorContext(ctx, "Failed to create email verification token", "error", err, "user_id", user.ID)
		return nil, "", fmt.Errorf("failed to create verification token: %w", err)
	}
	
	// Send verification email
	verifyURL := s.buildVerificationURL(verifyToken)
	if err := s.mailer.SendVerificationEmail(user.Email, user.Name, verifyURL, verifyToken); err != nil {
		logger.ErrorContext(ctx, "Failed to send verification email", "error", err, "user_id", user.ID)
		// Don't fail registration if email fails
	}
	
	return user, verifyURL, nil
}

func (s *authService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	// Normalize and validate
	req.Normalize()
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Find user
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Check if email is verified
	if !user.IsVerified {
		return nil, fmt.Errorf("email not verified")
	}
	
	// Verify password
	valid, err := argon2id.ComparePasswordAndHash(req.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Generate tokens
	scope := s.generateScope(user.Role)
	accessToken, err := auth.NewAccessToken(
		user.ID,
		user.Email,
		user.Role,
		scope,
		s.config.Auth.JWTSecret,
		s.config.Auth.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}
	
	// Create refresh token (longer lived)
	refreshToken, err := auth.NewAccessToken(
		user.ID,
		user.Email,
		"refresh",
		"refresh",
		s.config.Auth.JWTSecret,
		s.config.Auth.RefreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}
	
	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.Auth.AccessTokenTTL.Seconds()),
		User:         user.ToUserInfo(),
	}, nil
}

func (s *authService) VerifyEmail(ctx context.Context, token string) (*domain.User, error) {
	// Consume verification token
	userID, err := s.verifyRepo.ConsumeEmailVerification(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to consume verification token: %w", err)
	}
	if userID == 0 {
		return nil, fmt.Errorf("invalid or expired verification token")
	}
	
	// Mark user as verified
	if err := s.userRepo.MarkVerified(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to mark user as verified: %w", err)
	}
	
	// Get updated user
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get verified user: %w", err)
	}
	
	return user, nil
}

func (s *authService) ResendVerification(ctx context.Context, email string) error {
	// Find user
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		// Don't reveal if user exists or not
		return nil
	}
	
	// Check if already verified
	if user.IsVerified {
		return fmt.Errorf("account is already verified")
	}
	
	// Create new verification token
	verifyToken := uuid.NewString()
	expiresAt := time.Now().Add(s.config.Auth.EmailVerificationTTL)
	
	if err := s.verifyRepo.CreateEmailVerification(ctx, user.ID, verifyToken, expiresAt); err != nil {
		return fmt.Errorf("failed to create verification token: %w", err)
	}
	
	// Send verification email
	verifyURL := s.buildVerificationURL(verifyToken)
	if err := s.mailer.SendVerificationEmail(user.Email, user.Name, verifyURL, verifyToken); err != nil {
		logger.ErrorContext(ctx, "Failed to send verification email", "error", err, "user_id", user.ID)
		return fmt.Errorf("failed to send verification email: %w", err)
	}
	
	return nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResponse, error) {
	// Parse and validate refresh token
	claims, err := auth.Parse(refreshToken, s.config.Auth.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	
	if claims.Role != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}
	
	// Get user
	user, err := s.userRepo.FindByID(ctx, claims.Sub)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	
	// Generate new access token
	scope := s.generateScope(user.Role)
	accessToken, err := auth.NewAccessToken(
		user.ID,
		user.Email,
		user.Role,
		scope,
		s.config.Auth.JWTSecret,
		s.config.Auth.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}
	
	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Return same refresh token
		ExpiresIn:    int64(s.config.Auth.AccessTokenTTL.Seconds()),
		User:         user.ToUserInfo(),
	}, nil
}

func (s *authService) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *authService) UpdateUser(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	user, err := s.userRepo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return user, nil
}

func (s *authService) DeleteUser(ctx context.Context, id int64) error {
	err := s.userRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (s *authService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	users, err := s.userRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func (s *authService) UpdateUserRole(ctx context.Context, userID int64, role string) error {
	if !domain.IsValidRole(role) {
		return fmt.Errorf("invalid role: %s", role)
	}
	
	err := s.userRepo.UpdateRole(ctx, userID, role)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}

// Helper methods
func (s *authService) generateScope(role string) string {
	switch role {
	case domain.RoleAdmin:
		return "admin:read admin:write bookings:read bookings:write users:read users:write"
	case domain.RoleDispatcher:
		return "dispatch:read dispatch:write bookings:read bookings:write"
	case domain.RoleDriver:
		return "driver:read driver:write assignments:read assignments:write"
	case domain.RoleRider:
		return "bookings:read:self bookings:write:self"
	default:
		return ""
	}
}

func (s *authService) buildVerificationURL(token string) string {
	baseURL := "http://localhost:5173" // Should come from config
	return fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)
}