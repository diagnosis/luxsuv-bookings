package guest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/platform/auth"
	"github.com/diagnosis/luxsuv-bookings/internal/platform/mailer"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AccessHandler struct {
	Verify   postgres.VerifyRepo
	EmailSvc mailer.Service
}

func NewAccessHandler(verify postgres.VerifyRepo, emailSvc mailer.Service) *AccessHandler {
	return &AccessHandler{Verify: verify, EmailSvc: emailSvc}
}

func (h *AccessHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/request", h.request) // {email}
	r.Post("/verify", h.verify)   // {email, code}
	r.Post("/magic", h.magic)     // ?token=...
	return r
}

type requestIn struct {
	Email string `json:"email"`
}

func (h *AccessHandler) request(w http.ResponseWriter, r *http.Request) {
	var in requestIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Email == "" {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))

	// 6-digit code
	code := fmt.Sprintf("%06d", time.Now().UnixNano()%900000+100000)
	hashBytes, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	codeHash := string(hashBytes)

	// magic token
	magic := uuid.NewString()
	expires := time.Now().Add(15 * time.Minute)

	var ip net.IP
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = net.ParseIP(strings.TrimSpace(strings.Split(xff, ",")[0]))
	}

	if err := h.Verify.CreateGuestAccess(r.Context(), email, codeHash, magic, expires, ip); err != nil {
		http.Error(w, "could not create access", http.StatusInternalServerError)
		return
	}

	link := "http://localhost:5173/guest/access?token=" + magic
	_ = h.EmailSvc.SendGuestAccess(email, code, link)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "code sent"})
}

type verifyIn struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *AccessHandler) verify(w http.ResponseWriter, r *http.Request) {
	var in verifyIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Email == "" || in.Code == "" {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))

	ok, err := h.Verify.CheckGuestCode(r.Context(), email, in.Code)
	if err != nil || !ok {
		http.Error(w, "invalid or expired code", http.StatusUnauthorized)
		return
	}

	token, _ := auth.NewGuestSession(email, 30*time.Minute)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"session_token": token, "expires_in": int64(1800)})
}

func (h *AccessHandler) magic(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	email, ok, err := h.Verify.ConsumeGuestMagic(r.Context(), token)
	if err != nil || !ok {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	jwt, _ := auth.NewGuestSession(email, 30*time.Minute)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"session_token": jwt, "expires_in": int64(1800)})
}
