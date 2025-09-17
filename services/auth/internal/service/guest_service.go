package service

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/diagnosis/luxsuv-bookings/pkg/auth"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/events"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/mailer"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type GuestService interface {
	RequestAccess(ctx context.Context, req *domain.GuestAccessRequest, clientIP net.IP) error
	VerifyCode(ctx context.Context, req *domain.GuestAccessVerify) (*domain.GuestSessionResponse, error)
	VerifyMagicLink(ctx context.Context, token string) (*domain.GuestSessionResponse, error)
}

type guestService struct {
	verifyRepo repository.VerifyRepository
	userRepo   repository.UserRepository
	mailer     mailer.Service
	eventBus   events.EventBus
	config     *config.Config
}

func NewGuestService(
	verifyRepo repository.VerifyRepository,
	userRepo repository.UserRepository,
	mailer mailer.Service,
	eventBus events.EventBus,
	config *config.Config,
) GuestService {
	return &guestService{
		verifyRepo: verifyRepo,
		userRepo:   userRepo,
		mailer:     mailer,
		eventBus:   eventBus,
		config:     config,
	}
}

func (s *guestService) RequestAccess(ctx context.Context, req *domain.GuestAccessRequest, clientIP net.IP) error {
	// Normalize and validate
	req.Normalize()
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// Check if this email belongs to a registered user
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to check for existing user", "error", err, "email", req.Email)
	}
	if user != nil {
		return fmt.Errorf("this email is associated with a registered account. Please login with your password instead")
	}
	
	// Generate 6-digit code
	code := s.generateVerificationCode()
	
	// Hash the code
	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash verification code: %w", err)
	}
	
	// Generate magic link token
	magicToken := uuid.NewString()
	
	// Set expiration
	expiresAt := time.Now().Add(domain.CodeExpirationTime)
	
	// Store in database
	if err := s.verifyRepo.CreateGuestAccess(ctx, req.Email, string(codeHash), magicToken, expiresAt, clientIP); err != nil {
		return fmt.Errorf("failed to create guest access record: %w", err)
	}
	
	// Build magic link
	magicLink := s.buildMagicLink(magicToken)
	
	// Send email
	if err := s.mailer.SendGuestAccessEmail(req.Email, code, magicLink); err != nil {
		logger.ErrorContext(ctx, "Failed to send guest access email", "error", err, "email", req.Email)
		// Don't fail the request - code was created successfully
	}
	
	return nil
}

func (s *guestService) VerifyCode(ctx context.Context, req *domain.GuestAccessVerify) (*domain.GuestSessionResponse, error) {
	// Normalize and validate
	req.Normalize()
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Check if this email belongs to a registered user
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to check for existing user", "error", err, "email", req.Email)
	}
	if user != nil {
		return nil, fmt.Errorf("this email is associated with a registered account. Please login with your password instead")
	}
	
	// Verify the code
	valid, err := s.verifyRepo.CheckGuestCode(ctx, req.Email, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to verify code: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid or expired verification code")
	}
	
	// Generate guest session token
	token, err := auth.NewGuestSession(req.Email, s.config.Auth.JWTSecret, s.config.Auth.GuestSessionTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest session: %w", err)
	}
	
	return &domain.GuestSessionResponse{
		SessionToken: token,
		ExpiresIn:    int64(s.config.Auth.GuestSessionTTL.Seconds()),
	}, nil
}

func (s *guestService) VerifyMagicLink(ctx context.Context, token string) (*domain.GuestSessionResponse, error) {
	// Consume magic token
	email, valid, err := s.verifyRepo.ConsumeGuestMagic(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify magic link: %w", err)
	}
	if !valid {
		return nil, fmt.Errorf("invalid or expired magic link")
	}
	
	// Check if this email belongs to a registered user
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to check for existing user", "error", err, "email", email)
	}
	if user != nil {
		return nil, fmt.Errorf("this email is associated with a registered account. Please login with your password instead")
	}
	
	// Generate guest session token
	sessionToken, err := auth.NewGuestSession(email, s.config.Auth.JWTSecret, s.config.Auth.GuestSessionTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest session: %w", err)
	}
	
	return &domain.GuestSessionResponse{
		SessionToken: sessionToken,
		ExpiresIn:    int64(s.config.Auth.GuestSessionTTL.Seconds()),
	}, nil
}

// Helper methods
func (s *guestService) generateVerificationCode() string {
	// Generate a 6-digit code
	code := time.Now().UnixNano() % 900000 + 100000
	return fmt.Sprintf("%06d", code)
}

func (s *guestService) buildMagicLink(token string) string {
	baseURL := "http://localhost:5173" // Should come from config
	return fmt.Sprintf("%s/guest/access/magic?token=%s", baseURL, token)
}