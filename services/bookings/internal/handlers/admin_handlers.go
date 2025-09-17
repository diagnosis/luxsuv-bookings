package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/domain"
	"github.com/go-chi/chi/v5"
)

// ListAllBookings handles listing all bookings for admin
func (h *Handlers) ListAllBookings(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "admin" {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	limit, offset := parsePagination(r)
	
	// Parse status filter
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		if st, ok := domain.ParseBookingStatus(statusParam); ok {
			bookings, err := h.bookingService.ListBookingsByStatus(r.Context(), st, limit, offset)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Failed to retrieve bookings")
				return
			}
			h.writeBookingDTOs(w, bookings)
			return
		} else {
			writeError(w, http.StatusBadRequest, "Invalid status parameter")
			return
		}
	}

	// List all bookings
	bookings, err := h.bookingService.ListAllBookings(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve bookings")
		return
	}

	h.writeBookingDTOs(w, bookings)
}

// GetBooking handles getting a single booking for admin
func (h *Handlers) GetBooking(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "admin" {
		writeError(w, http.StatusForbidden, "Admin access required")
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
	if booking == nil {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	writeJSON(w, http.StatusOK, booking)
}

// UpdateBooking handles updating any booking for admin
func (h *Handlers) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "admin" {
		writeError(w, http.StatusForbidden, "Admin access required")
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

	// Admin can update any booking (no ownership check)
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

// CancelBooking handles canceling any booking for admin
func (h *Handlers) CancelBooking(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "admin" {
		writeError(w, http.StatusForbidden, "Admin access required")
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid booking ID")
		return
	}

	// Admin can cancel any booking regardless of time constraints
	success, err := h.bookingService.CancelBooking(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !success {
		writeError(w, http.StatusNotFound, "Booking not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper to write booking DTOs
func (h *Handlers) writeBookingDTOs(w http.ResponseWriter, bookings []domain.Booking) {
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