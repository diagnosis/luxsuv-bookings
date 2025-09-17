package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/domain"
	"github.com/go-chi/chi/v5"
)

// CreateGuestBooking handles guest booking creation
func (h *Handlers) CreateGuestBooking(w http.ResponseWriter, r *http.Request) {
	var req domain.BookingGuestReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Get idempotency key if provided
	idempotencyKey := r.Header.Get("Idempotency-Key")

	// Create booking
	booking, err := h.bookingService.CreateGuestBooking(r.Context(), &req, idempotencyKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return response
	response := domain.BookingGuestRes{
		ID:          booking.ID,
		ManageToken: booking.ManageToken,
		Status:      string(booking.Status),
		ScheduledAt: booking.ScheduledAt,
	}

	statusCode := http.StatusCreated
	if idempotencyKey != "" {
		// Check if this was an existing booking
		// For simplicity, we'll always return 201 for now
	}

	writeJSON(w, statusCode, response)
}

// ListGuestBookings handles listing bookings for authenticated guest
func (h *Handlers) ListGuestBookings(w http.ResponseWriter, r *http.Request) {
	claims := getGuestClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Guest session required")
		return
	}

	limit, offset := parsePagination(r)
	
	// Parse status filter
	var statusPtr *domain.BookingStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		if st, ok := domain.ParseBookingStatus(raw); ok {
			statusPtr = &st
		} else {
			writeError(w, http.StatusBadRequest, "Invalid status parameter")
			return
		}
	}

	// List bookings by email
	bookings, err := h.bookingService.ListBookingsByEmail(r.Context(), claims.Email, limit, offset, statusPtr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve bookings")
		return
	}

	// Convert to DTOs (excluding manage_token for security)
	dtos := make([]domain.BookingDTO, 0, len(bookings))
	for _, b := range bookings {
		dtos = append(dtos, domain.BookingDTO{
			ID:              b.ID,
			Status:          string(b.Status),
			RiderName:       b.RiderName,
			RiderEmail:      b.RiderEmail,
			RiderPhone:      b.RiderPhone,
			Pickup:          b.Pickup,
			Dropoff:         b.Dropoff,
			ScheduledAt:     b.ScheduledAt,
			Notes:           b.Notes,
			Passengers:      b.Passengers,
			Luggages:        b.Luggages,
			RideType:        string(b.RideType),
			DriverID:        b.DriverID,
			RescheduleCount: b.RescheduleCount,
			CreatedAt:       b.CreatedAt,
			UpdatedAt:       b.UpdatedAt,
			UserID:          b.UserID,
		})
	}

	writeJSON(w, http.StatusOK, dtos)
}

// GetGuestBooking handles getting a single booking
func (h *Handlers) GetGuestBooking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Check if manage_token is provided (public access)
	if token := r.URL.Query().Get("manage_token"); token != "" {
		booking, err := h.bookingService.GetBookingWithToken(r.Context(), id, token)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to retrieve booking")
			return
		}
		if booking == nil {
			writeError(w, http.StatusNotFound, "Booking not found")
			return
		}
		writeJSON(w, http.StatusOK, booking)
		return
	}

	// Session-based access
	claims := getGuestClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	booking, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve booking")
		return
	}
	if booking == nil || !booking.IsOwner(claims.Email) {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Return booking without manage_token for session-based access
	safeBooking := *booking
	safeBooking.ManageToken = ""
	writeJSON(w, http.StatusOK, safeBooking)
}

// UpdateGuestBooking handles updating a guest booking
func (h *Handlers) UpdateGuestBooking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	var patch domain.GuestPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Check if manage_token is provided (public access)
	if token := r.URL.Query().Get("manage_token"); token != "" {
		updated, err := h.bookingService.UpdateGuestBooking(r.Context(), id, token, patch)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if updated == nil {
			writeError(w, http.StatusNotFound, "Booking not found")
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	// Session-based access
	claims := getGuestClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get booking to verify ownership
	existing, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify booking ownership")
		return
	}
	if existing == nil || !existing.IsOwner(claims.Email) {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Update using manage token (we have ownership through session)
	updated, err := h.bookingService.UpdateGuestBooking(r.Context(), id, existing.ManageToken, patch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if updated == nil {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Remove manage_token from session-based responses
	safeBooking := *updated
	safeBooking.ManageToken = ""
	writeJSON(w, http.StatusOK, safeBooking)
}

// CancelGuestBooking handles canceling a guest booking
func (h *Handlers) CancelGuestBooking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Check if manage_token is provided (public access)
	if token := r.URL.Query().Get("manage_token"); token != "" {
		success, err := h.bookingService.CancelGuestBooking(r.Context(), id, token)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !success {
			writeError(w, http.StatusNotFound, "Booking not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Session-based access
	claims := getGuestClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get booking to verify ownership
	booking, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve booking")
		return
	}
	if booking == nil || !booking.IsOwner(claims.Email) {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	success, err := h.bookingService.CancelGuestBooking(r.Context(), id, booking.ManageToken)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !success {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}