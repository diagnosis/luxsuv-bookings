package guest

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/internal/http/response"
	"github.com/diagnosis/luxsuv-bookings/internal/http/middleware/guest_middleware"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/diagnosis/luxsuv-bookings/internal/utils"
	"github.com/go-chi/chi/v5"
)

type BookingsHandler struct{
	Repo           *postgres.BookingRepoImpl
	IdempotencyRepo postgres.IdempotencyRepo
}

func NewBookingsHandler(repo *postgres.BookingRepoImpl, idempotencyRepo postgres.IdempotencyRepo) *BookingsHandler {
	return &BookingsHandler{
		Repo:           repo,
		IdempotencyRepo: idempotencyRepo,
	}
}

func (h *BookingsHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.create)

	r.Group(func(pr chi.Router) { // list requires session
		pr.Use(guest_middleware.RequireGuestSession)
		pr.Get("/", h.list)
	})

	r.Group(func(pr chi.Router) { // optional: manage_token OR guest session
		pr.Use(guest_middleware.OptionalGuestSession)
		pr.Get("/{id}", h.getByID)
		pr.Patch("/{id}", h.patch)
		pr.Delete("/{id}", h.cancel)
	})

	return r
}

func (h *BookingsHandler) create(w http.ResponseWriter, r *http.Request) {
	// Handle idempotency
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey != "" {
		if existingBookingID, err := h.IdempotencyRepo.CheckOrCreateIdempotency(r.Context(), idempotencyKey, 0); err != nil {
			log.Printf("idempotency check failed: %v", err)
			response.InternalError(w, "Failed to check request uniqueness")
			return
		} else if existingBookingID > 0 {
			// Return existing booking
			if booking, err := h.Repo.GetByID(r.Context(), existingBookingID); err != nil || booking == nil {
				response.InternalError(w, "Failed to retrieve existing booking")
				return
			} else {
				out := domain.BookingGuestRes{
					ID:          booking.ID,
					ManageToken: booking.ManageToken,
					Status:      string(booking.Status),
					ScheduledAt: booking.ScheduledAt,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK) // Return 200 for idempotent response
				_ = json.NewEncoder(w).Encode(out)
				return
			}
		}
	}

	var in domain.BookingGuestReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "Invalid JSON format")
		return
	}

	// Normalize string inputs
	in.RiderName = utils.NormalizeString(in.RiderName)
	in.RiderEmail = utils.NormalizeEmail(in.RiderEmail)
	in.RiderPhone = utils.NormalizePhone(in.RiderPhone)
	in.Pickup = utils.NormalizeString(in.Pickup)
	in.Dropoff = utils.NormalizeString(in.Dropoff)
	in.Notes = utils.NormalizeString(in.Notes)

	// Validate required fields
	if in.RiderName == "" || in.RiderEmail == "" || in.RiderPhone == "" ||
		in.Pickup == "" || in.Dropoff == "" || in.ScheduledAt.IsZero() {
		response.WriteError(w, http.StatusBadRequest, "Missing required fields: rider_name, rider_email, rider_phone, pickup, dropoff, scheduled_at", response.CodeInvalidInput)
		return
	}

	// Validate email format
	if !utils.IsValidEmail(in.RiderEmail) {
		response.WriteError(w, http.StatusBadRequest, "Invalid email format", response.CodeInvalidInput)
		return
	}

	// Validate phone format
	if !utils.IsValidPhone(in.RiderPhone) {
		response.WriteError(w, http.StatusBadRequest, "Invalid phone number format", response.CodeInvalidInput)
		return
	}

	// Validate future date
	if in.ScheduledAt.Before(time.Now()) {
		response.WriteError(w, http.StatusBadRequest, "Scheduled time must be in the future", response.CodePastDateTime)
		return
	}

	// Validate numeric ranges
	if in.Passengers < 1 || in.Passengers > 8 {
		response.WriteError(w, http.StatusBadRequest, "Number of passengers must be between 1 and 8", response.CodeInvalidInput)
		return
	}
	if in.Luggages < 0 || in.Luggages > 10 {
		response.WriteError(w, http.StatusBadRequest, "Number of luggages must be between 0 and 10", response.CodeInvalidInput)
		return
	}
	if in.RideType != domain.RidePerRide && in.RideType != domain.RideHourly {
		response.WriteError(w, http.StatusBadRequest, "Ride type must be 'per_ride' or 'hourly'", response.CodeInvalidInput)
		return
	}

	b, err := h.Repo.CreateGuest(r.Context(), &in)
	if err != nil {
		log.Printf("failed to create guest booking: %v", err)
		response.InternalError(w, "Failed to create booking")
		return
	}

	// Store idempotency record if key was provided
	if idempotencyKey != "" {
		if _, err := h.IdempotencyRepo.CheckOrCreateIdempotency(r.Context(), idempotencyKey, b.ID); err != nil {
			log.Printf("failed to store idempotency record: %v", err)
			// Don't fail the request, just log the error
		}
	}

	out := domain.BookingGuestRes{ID: b.ID, ManageToken: b.ManageToken, Status: string(b.Status), ScheduledAt: b.ScheduledAt}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *BookingsHandler) list(w http.ResponseWriter, r *http.Request) {
	claims := guest_middleware.Claims(r)
	if claims == nil {
		response.Unauthorized(w, "Valid guest session required")
		return
	}

	limit, offset := 20, 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		} else {
			response.BadRequest(w, "Invalid limit parameter")
			return
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		} else {
			response.BadRequest(w, "Invalid offset parameter")
			return
		}
	}
	var statusPtr *domain.BookingStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		if st, ok := domain.ParseBookingStatus(raw); ok {
			statusPtr = &st
		} else {
			response.WriteError(w, http.StatusBadRequest, "Invalid status. Must be one of: pending, confirmed, assigned, on_trip, completed, canceled", response.CodeInvalidInput)
			return
		}
	}

	bs, err := h.Repo.ListByEmail(r.Context(), claims.Email, limit, offset, statusPtr)
	if err != nil {
		log.Printf("failed to list bookings by email: %v", err)
		response.InternalError(w, "Failed to retrieve bookings")
		return
	}

	// Convert to DTOs and ensure manage_token is not included
	out := make([]domain.BookingDTO, 0, len(bs))
	for _, b := range bs {
		out = append(out, domain.BookingDTO{
			ID: b.ID, Status: string(b.Status),
			RiderName: b.RiderName, RiderEmail: b.RiderEmail, RiderPhone: b.RiderPhone,
			Pickup: b.Pickup, Dropoff: b.Dropoff, ScheduledAt: b.ScheduledAt, Notes: b.Notes,
			Passengers: b.Passengers, Luggages: b.Luggages, RideType: string(b.RideType),
			DriverID: b.DriverID, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
			// Note: manage_token is intentionally excluded from BookingDTO
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *BookingsHandler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid booking ID")
		return
	}

	// Check if manage_token is provided (public access)
	if tok := r.URL.Query().Get("manage_token"); tok != "" {
		b, err := h.Repo.GetByIDWithToken(r.Context(), id, tok)
		if err != nil {
			log.Printf("failed to get booking by ID and token: %v", err)
			response.InternalError(w, "Failed to retrieve booking")
			return
		}
		if b == nil {
			response.NotFound(w, "Booking not found or invalid access token")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
		return
	}

	// Session-based access
	claims := guest_middleware.Claims(r)
	if claims == nil {
		response.Unauthorized(w, "Authentication required. Provide either manage_token or valid guest session")
		return
	}
	
	if claims.Role != "guest" {
		response.Forbidden(w, "Guest session required")
		return
	}

	b, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("failed to get booking by ID: %v", err)
		response.InternalError(w, "Failed to retrieve booking")
		return
	}
	if b == nil || strings.ToLower(b.RiderEmail) != strings.ToLower(claims.Email) {
		response.NotFound(w, "Booking not found")
		return
	}

	// For session-based access, return booking without manage_token for security
	safeBooking := *b
	safeBooking.ManageToken = "" // Remove manage token from session-based responses
	
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(safeBooking)
}

func (h *BookingsHandler) patch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid booking ID")
		return
	}

	var in domain.GuestPatch

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "Invalid JSON format")
		return
	}

	// Normalize string inputs
	if in.RiderName != nil {
		normalized := utils.NormalizeString(*in.RiderName)
		in.RiderName = &normalized
	}
	if in.RiderPhone != nil {
		normalized := utils.NormalizePhone(*in.RiderPhone)
		in.RiderPhone = &normalized
		if !utils.IsValidPhone(normalized) {
			response.WriteError(w, http.StatusBadRequest, "Invalid phone number format", response.CodeInvalidInput)
			return
		}
	}
	if in.Pickup != nil {
		normalized := utils.NormalizeString(*in.Pickup)
		in.Pickup = &normalized
	}
	if in.Dropoff != nil {
		normalized := utils.NormalizeString(*in.Dropoff)
		in.Dropoff = &normalized
	}
	if in.Notes != nil {
		normalized := utils.NormalizeString(*in.Notes)
		in.Notes = &normalized
	}

	// Validate fields if provided
	if in.ScheduledAt != nil && in.ScheduledAt.Before(time.Now()) {
		response.WriteError(w, http.StatusBadRequest, "Scheduled time must be in the future", response.CodePastDateTime)
		return
	}
	if in.Passengers != nil && (*in.Passengers < 1 || *in.Passengers > 8) {
		response.WriteError(w, http.StatusBadRequest, "Number of passengers must be between 1 and 8", response.CodeInvalidInput)
		return
	}
	if in.Luggages != nil && (*in.Luggages < 0 || *in.Luggages > 10) {
		response.WriteError(w, http.StatusBadRequest, "Number of luggages must be between 0 and 10", response.CodeInvalidInput)
		return
	}
	if in.RideType != nil && *in.RideType != domain.RidePerRide && *in.RideType != domain.RideHourly {
		response.WriteError(w, http.StatusBadRequest, "Ride type must be 'per_ride' or 'hourly'", response.CodeInvalidInput)
		return
	}

	// Check for manage_token (public access)
	if tok := r.URL.Query().Get("manage_token"); tok != "" {
		b, err := h.Repo.UpdateGuest(r.Context(), id, tok, in)
		if err != nil {
			log.Printf("failed to update guest booking: %v", err)
			response.InternalError(w, "Failed to update booking")
			return
		}
		if b == nil {
			response.NotFound(w, "Booking not found or invalid access token")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
		return
	}

	// Session-based access
	claims := guest_middleware.Claims(r)
	if claims == nil {
		response.Unauthorized(w, "Authentication required. Provide either manage_token or valid guest session")
		return
	}

	if claims.Role != "guest" {
		response.Forbidden(w, "Guest session required")
		return
	}

	// Get booking to verify ownership
	existing, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("failed to get booking for ownership check: %v", err)
		response.InternalError(w, "Failed to verify booking ownership")
		return
	}
	if existing == nil || strings.ToLower(existing.RiderEmail) != strings.ToLower(claims.Email) {
		response.NotFound(w, "Booking not found")
		return
	}

	// Update using manage token (we have ownership through session)
	b, err := h.Repo.UpdateGuest(r.Context(), id, existing.ManageToken, in)
	if err != nil {
		log.Printf("failed to update booking via session: %v", err)
		response.InternalError(w, "Failed to update booking")
		return
	}
	if b == nil {
		response.NotFound(w, "Booking not found")
		return
	}

	// Remove manage_token from session-based responses
	safeBooking := *b
	safeBooking.ManageToken = ""

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(safeBooking)
}

func (h *BookingsHandler) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid booking ID")
		return
	}

	// Check for manage_token (public access)
	if tok := r.URL.Query().Get("manage_token"); tok != "" {
		ok, err := h.Repo.CancelWithToken(r.Context(), id, tok)
		if err != nil {
			log.Printf("failed to cancel booking with token: %v", err)
			response.InternalError(w, "Failed to cancel booking")
			return
		}
		if !ok {
			response.NotFound(w, "Booking not found or invalid access token")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Session-based access
	claims := guest_middleware.Claims(r)
	if claims == nil {
		response.Unauthorized(w, "Authentication required. Provide either manage_token or valid guest session")
		return
	}

	if claims.Role != "guest" {
		response.Forbidden(w, "Guest session required")
		return
	}

	b, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("failed to get booking for cancellation: %v", err)
		response.InternalError(w, "Failed to retrieve booking")
		return
	}
	if b == nil || strings.ToLower(b.RiderEmail) != strings.ToLower(claims.Email) {
		response.NotFound(w, "Booking not found")
		return
	}

	ok, err := h.Repo.CancelWithToken(r.Context(), id, b.ManageToken)
	if err != nil {
		log.Printf("failed to cancel booking: %v", err)
		response.InternalError(w, "Failed to cancel booking")
		return
	}
	if !ok {
		response.NotFound(w, "Booking not found or already canceled")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
