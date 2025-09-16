package guest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/internal/http/middleware/guest_middleware"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/go-chi/chi/v5"
)

type BookingsHandler struct{ Repo *postgres.BookingRepoImpl }

func NewBookingsHandler(repo *postgres.BookingRepoImpl) *BookingsHandler {
	return &BookingsHandler{Repo: repo}
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
	var in domain.BookingGuestReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if in.RiderName == "" || in.RiderEmail == "" || in.RiderPhone == "" ||
		in.Pickup == "" || in.Dropoff == "" || in.ScheduledAt.IsZero() {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	if in.ScheduledAt.Before(time.Now()) {
		http.Error(w, "scheduled_at must be future", http.StatusBadRequest)
		return
	}
	if in.Passengers < 1 || in.Passengers > 8 {
		http.Error(w, "invalid passengers", http.StatusBadRequest)
		return
	}
	if in.Luggages < 0 || in.Luggages > 10 {
		http.Error(w, "invalid luggages", http.StatusBadRequest)
		return
	}
	if in.RideType != domain.RidePerRide && in.RideType != domain.RideHourly {
		http.Error(w, "ride_type must be 'per_ride' or 'hourly'", http.StatusBadRequest)
		return
	}

	b, err := h.Repo.CreateGuest(r.Context(), &in)
	if err != nil {
		http.Error(w, "error creating", http.StatusInternalServerError)
		return
	}

	out := domain.BookingGuestRes{ID: b.ID, ManageToken: b.ManageToken, Status: string(b.Status), ScheduledAt: b.ScheduledAt}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *BookingsHandler) list(w http.ResponseWriter, r *http.Request) {
	claims := guest_middleware.Claims(r)
	limit, offset := 20, 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	var statusPtr *domain.BookingStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		if st, ok := domain.ParseBookingStatus(raw); ok {
			statusPtr = &st
		} else {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}
	}

	bs, err := h.Repo.ListByEmail(r.Context(), claims.Email, limit, offset, statusPtr)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	out := make([]domain.BookingDTO, 0, len(bs))
	for _, b := range bs {
		out = append(out, domain.BookingDTO{
			ID: b.ID, Status: string(b.Status),
			RiderName: b.RiderName, RiderEmail: b.RiderEmail, RiderPhone: b.RiderPhone,
			Pickup: b.Pickup, Dropoff: b.Dropoff, ScheduledAt: b.ScheduledAt, Notes: b.Notes,
			Passengers: b.Passengers, Luggages: b.Luggages, RideType: string(b.RideType),
			DriverID: b.DriverID, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *BookingsHandler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	if tok := r.URL.Query().Get("manage_token"); tok != "" {
		b, err := h.Repo.GetByIDWithToken(r.Context(), id, tok)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if b == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
		return
	}

	claims := guest_middleware.Claims(r)
	if claims == nil || claims.Role != "guest" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	b, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if b == nil || strings.ToLower(b.RiderEmail) != strings.ToLower(claims.Email) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

func (h *BookingsHandler) patch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	var in domain.GuestPatch

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if in.ScheduledAt != nil && in.ScheduledAt.Before(time.Now()) {
		http.Error(w, "scheduled_at must be future", http.StatusBadRequest)
		return
	}
	if in.Passengers != nil && (*in.Passengers < 1 || *in.Passengers > 8) {
		http.Error(w, "invalid passengers", http.StatusBadRequest)
		return
	}
	if in.Luggages != nil && (*in.Luggages < 0 || *in.Luggages > 10) {
		http.Error(w, "invalid luggages", http.StatusBadRequest)
		return
	}
	if in.RideType != nil && *in.RideType != domain.RidePerRide && *in.RideType != domain.RideHourly {
		http.Error(w, "ride_type must be 'per_ride' or 'hourly'", http.StatusBadRequest)
		return
	}

	if tok := r.URL.Query().Get("manage_token"); tok != "" {
		b, err := h.Repo.UpdateGuest(r.Context(), id, tok, in)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if b == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
		return
	}

	claims := guest_middleware.Claims(r)
	if claims == nil || claims.Role != "guest" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h *BookingsHandler) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	if tok := r.URL.Query().Get("manage_token"); tok != "" {
		ok, err := h.Repo.CancelWithToken(r.Context(), id, tok)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	claims := guest_middleware.Claims(r)
	if claims == nil || claims.Role != "guest" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	b, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if b == nil || strings.ToLower(b.RiderEmail) != strings.ToLower(claims.Email) {
		http.NotFound(w, r)
		return
	}

	ok, err := h.Repo.CancelWithToken(r.Context(), id, b.ManageToken)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
