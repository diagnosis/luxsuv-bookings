package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/domain"
	"github.com/go-chi/chi/v5"
)

// CreateRiderBooking handles rider booking creation
func (h *Handlers) CreateRiderBooking(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var req domain.BookingGuestReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Create booking for authenticated rider
	booking, err := h.bookingService.CreateRiderBooking(r.Context(), claims.Sub, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to DTO
	dto := domain.BookingDTO{
		ID:              booking.ID,
		Status:          string(booking.Status),
		RiderName:       booking.RiderName,
		RiderEmail:      booking.RiderEmail,
		RiderPhone:      booking.RiderPhone,
		Pickup:          booking.Pickup,
		Dropoff:         booking.Dropoff,
		ScheduledAt:     booking.ScheduledAt,
		Notes:           booking.Notes,
		Passengers:      booking.Passengers,
		Luggages:        booking.Luggages,
		RideType:        string(booking.RideType),
		DriverID:        booking.DriverID,
		RescheduleCount: booking.RescheduleCount,
		CreatedAt:       booking.CreatedAt,
		UpdatedAt:       booking.UpdatedAt,
		UserID:          booking.UserID,
	}

	writeJSON(w, http.StatusCreated, dto)
}

// ListRiderBookings handles listing bookings for authenticated rider
func (h *Handlers) ListRiderBookings(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "rider" {
		writeError(w, http.StatusForbidden, "Rider access required")
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

	// List bookings by user ID
	bookings, err := h.bookingService.ListBookingsByUser(r.Context(), claims.Sub, limit, offset, statusPtr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve bookings")
		return
	}

	// Convert to DTOs
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

// GetRiderBooking handles getting a single booking for rider
func (h *Handlers) GetRiderBooking(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "rider" {
		writeError(w, http.StatusForbidden, "Rider access required")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	booking, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve booking")
		return
	}
	if booking == nil || !booking.IsUserOwner(claims.Sub) {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	writeJSON(w, http.StatusOK, booking)
}

// UpdateRiderBooking handles updating a rider booking
func (h *Handlers) UpdateRiderBooking(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "rider" {
		writeError(w, http.StatusForbidden, "Rider access required")
		return
	}

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

	// Check ownership
	booking, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify booking ownership")
		return
	}
	if booking == nil || !booking.IsUserOwner(claims.Sub) {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Update booking
	updated, err := h.bookingService.UpdateBooking(r.Context(), id, patch)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if updated == nil {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// CancelRiderBooking handles canceling a rider booking
func (h *Handlers) CancelRiderBooking(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "rider" {
		writeError(w, http.StatusForbidden, "Rider access required")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Check ownership
	booking, err := h.bookingService.GetBooking(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve booking")
		return
	}
	if booking == nil || !booking.IsUserOwner(claims.Sub) {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	// Use guest cancellation method which enforces 24h rule
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