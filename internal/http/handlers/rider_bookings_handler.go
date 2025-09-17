package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/domain"
	mw "github.com/diagnosis/luxsuv-bookings/internal/http/middleware"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/go-chi/chi/v5"
)

type RiderBookingsHandler struct {
	Bookings *postgres.BookingRepoImpl
	Users    postgres.UsersRepo
}

func NewRiderBookingsHandler(b *postgres.BookingRepoImpl, u postgres.UsersRepo) *RiderBookingsHandler {
	return &RiderBookingsHandler{Bookings: b, Users: u}
}

func (h *RiderBookingsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(mw.RequireJWT)
	r.Get("/", h.list)
	r.Get("/{id}", h.getByID)
	r.Delete("/{id}", h.cancel)
	r.Post("/", h.create)
	return r
}

type riderCreateReq struct {
	Pickup      string          `json:"pickup"`
	Dropoff     string          `json:"dropoff"`
	ScheduledAt time.Time       `json:"scheduled_at"`
	Notes       string          `json:"notes"`
	Passengers  int             `json:"passengers"`
	Luggages    int             `json:"luggages"`
	RideType    domain.RideType `json:"ride_type"`
}

func (h *RiderBookingsHandler) create(w http.ResponseWriter, r *http.Request) {
	claims := mw.Claims(r)
	if claims == nil || claims.Sub == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var in riderCreateReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	// minimal validation
	if in.Pickup == "" || in.Dropoff == "" || in.ScheduledAt.IsZero() {
		http.Error(w, "pickup/dropoff/scheduled_at required", http.StatusBadRequest)
		return
	}
	if in.ScheduledAt.Before(time.Now()) {
		http.Error(w, "scheduled_at must be future", http.StatusBadRequest)
		return
	}
	if in.Passengers <= 0 || in.Passengers > 8 {
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

	// fetch rider contact
	u, err := h.Users.FindByEmail(r.Context(), claims.Email)
	if err != nil || u == nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	b, err := h.Bookings.CreateForUser(r.Context(), claims.Sub, &domain.BookingGuestReq{
		RiderName:   u.Name,
		RiderEmail:  u.Email,
		RiderPhone:  u.Phone,
		Pickup:      in.Pickup,
		Dropoff:     in.Dropoff,
		ScheduledAt: in.ScheduledAt,
		Notes:       in.Notes,
		Passengers:  in.Passengers,
		Luggages:    in.Luggages,
		RideType:    in.RideType,
	})
	if err != nil {
		http.Error(w, "could not create booking", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(domain.BookingDTO{
		ID: b.ID, Status: string(b.Status),
		RiderName: b.RiderName, RiderEmail: b.RiderEmail, RiderPhone: b.RiderPhone,
		Pickup: b.Pickup, Dropoff: b.Dropoff, ScheduledAt: b.ScheduledAt, Notes: b.Notes,
		Passengers: b.Passengers, Luggages: b.Luggages, RideType: string(b.RideType),
		DriverID: b.DriverID, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	})
}

func (h *RiderBookingsHandler) list(w http.ResponseWriter, r *http.Request) {
	claims := mw.Claims(r)
	if claims == nil || claims.Role != "rider" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
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

	bs, err := h.Bookings.ListByUserID(r.Context(), claims.Sub, limit, offset, statusPtr)
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

func (h *RiderBookingsHandler) getByID(w http.ResponseWriter, r *http.Request) {
	claims := mw.Claims(r)
	if claims == nil || claims.Role != "rider" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	b, err := h.Bookings.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if b == nil || b.RiderEmail != claims.Email && b.DriverID != &claims.Sub && b.ID == 0 {
		// simplest ownership check for rider: by user_id (preferred) or fallback by email if you haven’t backfilled
		if b == nil || b.RiderEmail != claims.Email {
			http.NotFound(w, r)
			return
		}
	}
	// optional: if you already set user_id on bookings, check that instead of email
	// (add user_id to Booking struct if you want)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

func (h *RiderBookingsHandler) cancel(w http.ResponseWriter, r *http.Request) {
	claims := mw.Claims(r)
	if claims == nil || claims.Role != "rider" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}

	b, err := h.Bookings.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if b == nil {
		http.NotFound(w, r)
		return
	}
	// ownership: prefer user_id check; fallback to email
	if b.RiderEmail != claims.Email {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Soft cancel (reuse your repo CancelWithToken, or add a CancelByID)
	if _, err := h.Bookings.CancelWithToken(r.Context(), id, b.ManageToken); err != nil {
		http.Error(w, "cancel error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	_ = time.Now()
}
