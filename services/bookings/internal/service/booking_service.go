package service

import (
	"context"
	"fmt"
	"time"

	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/events"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/repository"
)

type BookingService interface {
	CreateGuestBooking(ctx context.Context, req *domain.BookingGuestReq, idempotencyKey string) (*domain.Booking, error)
	CreateRiderBooking(ctx context.Context, userID int64, req *domain.BookingGuestReq) (*domain.Booking, error)
	GetBooking(ctx context.Context, id int64) (*domain.Booking, error)
	GetBookingWithToken(ctx context.Context, id int64, token string) (*domain.Booking, error)
	ListBookingsByEmail(ctx context.Context, email string, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error)
	ListBookingsByUser(ctx context.Context, userID int64, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error)
	ListAllBookings(ctx context.Context, limit, offset int) ([]domain.Booking, error)
	ListBookingsByStatus(ctx context.Context, status domain.BookingStatus, limit, offset int) ([]domain.Booking, error)
	UpdateGuestBooking(ctx context.Context, id int64, token string, patch domain.GuestPatch) (*domain.Booking, error)
	UpdateBooking(ctx context.Context, id int64, patch domain.GuestPatch) (*domain.Booking, error)
	CancelGuestBooking(ctx context.Context, id int64, token string) (bool, error)
	CancelBooking(ctx context.Context, id int64) (bool, error)
}

type bookingService struct {
	bookingRepo     repository.BookingRepository
	idempotencyRepo repository.IdempotencyRepository
	userRepo        repository.UserRepository
	eventBus        events.EventBus
	config          *config.Config
}

func NewBookingService(
	bookingRepo repository.BookingRepository,
	idempotencyRepo repository.IdempotencyRepository,
	userRepo repository.UserRepository,
	eventBus events.EventBus,
	config *config.Config,
) BookingService {
	return &bookingService{
		bookingRepo:     bookingRepo,
		idempotencyRepo: idempotencyRepo,
		userRepo:        userRepo,
		eventBus:        eventBus,
		config:          config,
	}
}

func (s *bookingService) CreateGuestBooking(ctx context.Context, req *domain.BookingGuestReq, idempotencyKey string) (*domain.Booking, error) {
	// Validate business rules
	if err := s.validateBookingRequest(req); err != nil {
		return nil, err
	}

	// Check idempotency if key provided
	if idempotencyKey != "" {
		if existingBookingID, err := s.idempotencyRepo.CheckOrCreateIdempotency(ctx, idempotencyKey, 0); err != nil {
			return nil, fmt.Errorf("idempotency check failed: %w", err)
		} else if existingBookingID > 0 {
			// Return existing booking
			return s.bookingRepo.GetByID(ctx, existingBookingID)
		}
	}

	// Create booking
	booking, err := s.bookingRepo.CreateGuest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create guest booking: %w", err)
	}

	// Store idempotency record if key was provided
	if idempotencyKey != "" {
		if _, err := s.idempotencyRepo.CheckOrCreateIdempotency(ctx, idempotencyKey, booking.ID); err != nil {
			logger.ErrorContext(ctx, "Failed to store idempotency record", "error", err, "booking_id", booking.ID)
		}
	}

	// Publish booking created event
	event := events.BookingCreatedEvent{
		BookingID:   booking.ID,
		RiderEmail:  booking.RiderEmail,
		RiderName:   booking.RiderName,
		Pickup:      booking.Pickup,
		Dropoff:     booking.Dropoff,
		ScheduledAt: booking.ScheduledAt,
		Passengers:  booking.Passengers,
		CreatedAt:   booking.CreatedAt,
	}

	if err := s.eventBus.Publish(ctx, events.BookingCreated, event); err != nil {
		logger.ErrorContext(ctx, "Failed to publish booking created event", "error", err, "booking_id", booking.ID)
	}

	return booking, nil
}

func (s *bookingService) CreateRiderBooking(ctx context.Context, userID int64, req *domain.BookingGuestReq) (*domain.Booking, error) {
	// Validate business rules
	if err := s.validateBookingRequest(req); err != nil {
		return nil, err
	}

	// Get user details for the booking
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Use user's details for the booking
	req.RiderName = user.Name
	req.RiderEmail = user.Email
	req.RiderPhone = user.Phone

	// Create booking
	booking, err := s.bookingRepo.CreateForUser(ctx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create rider booking: %w", err)
	}

	// Publish booking created event
	event := events.BookingCreatedEvent{
		BookingID:   booking.ID,
		RiderEmail:  booking.RiderEmail,
		RiderName:   booking.RiderName,
		Pickup:      booking.Pickup,
		Dropoff:     booking.Dropoff,
		ScheduledAt: booking.ScheduledAt,
		Passengers:  booking.Passengers,
		CreatedAt:   booking.CreatedAt,
	}

	if err := s.eventBus.Publish(ctx, events.BookingCreated, event); err != nil {
		logger.ErrorContext(ctx, "Failed to publish booking created event", "error", err, "booking_id", booking.ID)
	}

	return booking, nil
}

func (s *bookingService) GetBooking(ctx context.Context, id int64) (*domain.Booking, error) {
	return s.bookingRepo.GetByID(ctx, id)
}

func (s *bookingService) GetBookingWithToken(ctx context.Context, id int64, token string) (*domain.Booking, error) {
	return s.bookingRepo.GetByIDWithToken(ctx, id, token)
}

func (s *bookingService) ListBookingsByEmail(ctx context.Context, email string, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error) {
	return s.bookingRepo.ListByEmail(ctx, email, limit, offset, status)
}

func (s *bookingService) ListBookingsByUser(ctx context.Context, userID int64, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error) {
	return s.bookingRepo.ListByUserID(ctx, userID, limit, offset, status)
}

func (s *bookingService) ListAllBookings(ctx context.Context, limit, offset int) ([]domain.Booking, error) {
	return s.bookingRepo.List(ctx, limit, offset)
}

func (s *bookingService) ListBookingsByStatus(ctx context.Context, status domain.BookingStatus, limit, offset int) ([]domain.Booking, error) {
	return s.bookingRepo.ListByStatus(ctx, status, limit, offset)
}

func (s *bookingService) UpdateGuestBooking(ctx context.Context, id int64, token string, patch domain.GuestPatch) (*domain.Booking, error) {
	// Get existing booking to check business rules
	existing, err := s.bookingRepo.GetByIDWithToken(ctx, id, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("booking not found")
	}

	// Validate business rules
	if err := s.validateBookingUpdate(existing, patch); err != nil {
		return nil, err
	}

	// Update booking
	updated, err := s.bookingRepo.UpdateGuest(ctx, id, token, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	// Publish booking updated event if there were changes
	if updated != nil {
		changes := s.detectChanges(existing, updated)
		if len(changes) > 0 {
			event := events.BookingUpdatedEvent{
				BookingID:  updated.ID,
				RiderEmail: updated.RiderEmail,
				Changes:    changes,
				UpdatedAt:  updated.UpdatedAt,
			}

			if err := s.eventBus.Publish(ctx, events.BookingUpdated, event); err != nil {
				logger.ErrorContext(ctx, "Failed to publish booking updated event", "error", err, "booking_id", updated.ID)
			}
		}
	}

	return updated, nil
}

func (s *bookingService) UpdateBooking(ctx context.Context, id int64, patch domain.GuestPatch) (*domain.Booking, error) {
	// Get existing booking to check business rules
	existing, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("booking not found")
	}

	// Validate business rules
	if err := s.validateBookingUpdate(existing, patch); err != nil {
		return nil, err
	}

	// Update booking
	updated, err := s.bookingRepo.Update(ctx, id, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	// Publish booking updated event if there were changes
	if updated != nil {
		changes := s.detectChanges(existing, updated)
		if len(changes) > 0 {
			event := events.BookingUpdatedEvent{
				BookingID:  updated.ID,
				RiderEmail: updated.RiderEmail,
				Changes:    changes,
				UpdatedAt:  updated.UpdatedAt,
			}

			if err := s.eventBus.Publish(ctx, events.BookingUpdated, event); err != nil {
				logger.ErrorContext(ctx, "Failed to publish booking updated event", "error", err, "booking_id", updated.ID)
			}
		}
	}

	return updated, nil
}

func (s *bookingService) CancelGuestBooking(ctx context.Context, id int64, token string) (bool, error) {
	// Get booking to check business rules
	booking, err := s.bookingRepo.GetByIDWithToken(ctx, id, token)
	if err != nil {
		return false, fmt.Errorf("failed to get booking: %w", err)
	}
	if booking == nil {
		return false, fmt.Errorf("booking not found")
	}

	// Check if cancellation is allowed (24h rule)
	if !booking.CanCancel() {
		return false, fmt.Errorf("cannot cancel booking less than 24 hours before scheduled time")
	}

	// Cancel booking
	success, err := s.bookingRepo.CancelWithToken(ctx, id, token)
	if err != nil {
		return false, fmt.Errorf("failed to cancel booking: %w", err)
	}

	if success {
		// Publish booking canceled event
		event := events.BookingCanceledEvent{
			BookingID:  booking.ID,
			RiderEmail: booking.RiderEmail,
			Reason:     "user_requested",
			CanceledAt: time.Now(),
		}

		if err := s.eventBus.Publish(ctx, events.BookingCanceled, event); err != nil {
			logger.ErrorContext(ctx, "Failed to publish booking canceled event", "error", err, "booking_id", booking.ID)
		}
	}

	return success, nil
}

func (s *bookingService) CancelBooking(ctx context.Context, id int64) (bool, error) {
	// Get booking to check business rules
	booking, err := s.bookingRepo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to get booking: %w", err)
	}
	if booking == nil {
		return false, fmt.Errorf("booking not found")
	}

	// Admin/dispatcher can cancel regardless of time constraints
	// Cancel booking
	success, err := s.bookingRepo.Cancel(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to cancel booking: %w", err)
	}

	if success {
		// Publish booking canceled event
		event := events.BookingCanceledEvent{
			BookingID:  booking.ID,
			RiderEmail: booking.RiderEmail,
			Reason:     "admin_canceled",
			CanceledAt: time.Now(),
		}

		if err := s.eventBus.Publish(ctx, events.BookingCanceled, event); err != nil {
			logger.ErrorContext(ctx, "Failed to publish booking canceled event", "error", err, "booking_id", booking.ID)
		}
	}

	return success, nil
}

func (s *bookingService) validateBookingRequest(req *domain.BookingGuestReq) error {
	if req.Passengers < domain.MinPassengers || req.Passengers > domain.MaxPassengers {
		return fmt.Errorf("passengers must be between %d and %d", domain.MinPassengers, domain.MaxPassengers)
	}
	if req.Luggages < domain.MinLuggages || req.Luggages > domain.MaxLuggages {
		return fmt.Errorf("luggages must be between %d and %d", domain.MinLuggages, domain.MaxLuggages)
	}
	if req.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduled time must be in the future")
	}
	return nil
}

func (s *bookingService) validateBookingUpdate(existing *domain.Booking, patch domain.GuestPatch) error {
	// Check reschedule limit if scheduled_at is being changed
	if patch.ScheduledAt != nil && !patch.ScheduledAt.Equal(existing.ScheduledAt) {
		if !existing.CanReschedule() {
			return fmt.Errorf("maximum reschedule limit (%d) reached or booking cannot be rescheduled", domain.MaxRescheduleCount)
		}
	}

	// Validate new values
	if patch.Passengers != nil && (*patch.Passengers < domain.MinPassengers || *patch.Passengers > domain.MaxPassengers) {
		return fmt.Errorf("passengers must be between %d and %d", domain.MinPassengers, domain.MaxPassengers)
	}
	if patch.Luggages != nil && (*patch.Luggages < domain.MinLuggages || *patch.Luggages > domain.MaxLuggages) {
		return fmt.Errorf("luggages must be between %d and %d", domain.MinLuggages, domain.MaxLuggages)
	}
	if patch.ScheduledAt != nil && patch.ScheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduled time must be in the future")
	}

	return nil
}

func (s *bookingService) detectChanges(old, new *domain.Booking) []string {
	var changes []string

	if old.RiderName != new.RiderName {
		changes = append(changes, "rider_name")
	}
	if old.RiderPhone != new.RiderPhone {
		changes = append(changes, "rider_phone")
	}
	if old.Pickup != new.Pickup {
		changes = append(changes, "pickup")
	}
	if old.Dropoff != new.Dropoff {
		changes = append(changes, "dropoff")
	}
	if !old.ScheduledAt.Equal(new.ScheduledAt) {
		changes = append(changes, "scheduled_at")
	}
	if old.Notes != new.Notes {
		changes = append(changes, "notes")
	}
	if old.Passengers != new.Passengers {
		changes = append(changes, "passengers")
	}
	if old.Luggages != new.Luggages {
		changes = append(changes, "luggages")
	}
	if old.RideType != new.RideType {
		changes = append(changes, "ride_type")
	}

	return changes
}