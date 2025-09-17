package service

import (
	"context"
	"fmt"

	"github.com/diagnosis/luxsuv-bookings/pkg/auth"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/events"
	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/repository"
)

type GuestService interface {
	CreateGuestSession(ctx context.Context, email string) (string, int64, error)
}

type guestService struct {
	verifyRepo repository.VerifyRepository
	userRepo   repository.UserRepository
	eventBus   events.EventBus
	config     *config.Config
}

func NewGuestService(
	verifyRepo repository.VerifyRepository,
	userRepo repository.UserRepository,
	eventBus events.EventBus,
	config *config.Config,
) GuestService {
	return &guestService{
		verifyRepo: verifyRepo,
		userRepo:   userRepo,
		eventBus:   eventBus,
		config:     config,
	}
}

func (s *guestService) CreateGuestSession(ctx context.Context, email string) (string, int64, error) {
	// Check if this email belongs to a registered user
	if user, err := s.userRepo.FindByEmail(ctx, email); err == nil && user != nil {
		return "", 0, fmt.Errorf("this email is associated with a registered account")
	}

	// Create guest session token
	ttl := s.config.Auth.GuestSessionTTL
	token, err := auth.NewGuestSession(email, s.config.Auth.JWTSecret, ttl)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create guest session: %w", err)
	}

	return token, int64(ttl.Seconds()), nil
}
