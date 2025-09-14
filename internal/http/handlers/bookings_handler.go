package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/go-chi/chi/v5"
)

type BookingGuestHandler struct {
	Repo postgres.BookingRepo
}

func NewBookingGuestHandler(repo postgres.BookingRepo) *BookingGuestHandler {
	return &BookingGuestHandler{Repo: repo}
}

func (h *BookingGuestHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{id}", h.getByID)
	r.Delete("/{id}", h.cancel)
	return r
}

func (h *BookingGuestHandler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.BookingGuestReq
	err := json.NewDecoder(r.Body).Decode(&in)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		log.Println("error: " + err.Error())
		return
	}
	//basic validation
	if in.RiderName == "" || in.RiderEmail == "" || in.RiderPhone == "" || in.Pickup == "" || in.Dropoff == "" || in.ScheduledAt.IsZero() {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	if in.ScheduledAt.Before(time.Now().Add(-2 * time.Hour)) {
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
	//insert booking into db\
	b, err := h.Repo.CreateGuest(r.Context(), &in)
	if err != nil {
		http.Error(w, "error creating booking", http.StatusInternalServerError)
		log.Println("error: " + err.Error())
		return
	}

	out := domain.BookingGuestRes{
		ID:          b.ID,
		ManageToken: b.ManageToken,
		Status:      string(b.Status),
		ScheduledAt: b.ScheduledAt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *BookingGuestHandler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	token := r.URL.Query().Get("manage_token")
	if token == "" {
		http.Error(w, "manage_token is required", http.StatusUnauthorized)
		return
	}
	b, err := h.Repo.GetByIDWithToken(r.Context(), id, token)
	if err != nil {
		http.Error(w, "error getting booking", http.StatusInternalServerError)
		log.Println("error: " + err.Error())
		return
	}
	if b == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(b)
}

func (h *BookingGuestHandler) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	token := r.URL.Query().Get("manage_token")
	if token == "" {
		http.Error(w, "manage_token is required", http.StatusUnauthorized)
		return
	}
	ok, err := h.Repo.CancelWithToken(r.Context(), id, token)
	if err != nil {
		http.Error(w, "error cancelling booking", http.StatusInternalServerError)
		log.Println("error: " + err.Error())
		return
	}
	if !ok {
		http.NotFound(w, r)
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *BookingGuestHandler) list(w http.ResponseWriter, r *http.Request) {
	//defaults
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if n > 100 {
			n = 100
		}
		limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = n
	}
	var (
		bs  []domain.Booking
		err error
	)
	if raw := r.URL.Query().Get("status"); raw != "" {
		st, ok := domain.ParseBookingStatus(raw)
		if !ok {
			http.Error(w, "invalid status (allowed: pending, confirmed, assigned, on_trip, completed, canceled)", http.StatusBadRequest)
			return
		}
		bs, err = h.Repo.ListByStatus(r.Context(), st, limit, offset)
	} else {
		bs, err = h.Repo.List(r.Context(), limit, offset)
	}
	if err != nil {
		http.Error(w, "error listing bookings", http.StatusInternalServerError)
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
