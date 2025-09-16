package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/diagnosis/luxsuv-bookings/internal/platform/auth"
	"github.com/diagnosis/luxsuv-bookings/internal/platform/mailer"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AuthHandler struct {
	Users    postgres.UsersRepo
	Verify   postgres.VerifyRepo
	EmailSvc mailer.Service
}

func NewAuthHandler(users postgres.UsersRepo, verify postgres.VerifyRepo, emailSvc mailer.Service) *AuthHandler {
	return &AuthHandler{Users: users, Verify: verify, EmailSvc: emailSvc}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/verify-email", h.verifyEmail) // POST ?token=...
	return r
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
		Phone    string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil ||
		in.Email == "" || in.Password == "" || in.Name == "" || in.Phone == "" {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	hash, err := argon2id.CreateHash(in.Password, argon2id.DefaultParams)
	if err != nil {
		http.Error(w, "hash error", http.StatusInternalServerError)
		return
	}

	u, err := h.Users.Create(r.Context(), email, hash, in.Name, in.Phone)
	if err != nil {
		// likely duplicate email or db error
		http.Error(w, "email exists or db error", http.StatusBadRequest)
		return
	}

	// Link historical bookings (same email)
	_ = h.Users.LinkExistingBookings(r.Context(), u.ID, email)

	// Create verification token (24h) and email it
	vtok := uuid.NewString()
	_ = h.Verify.CreateEmailVerification(r.Context(), u.ID, vtok, time.Now().Add(24*time.Hour))

	verifyURL := "http://localhost:5173/verify-email?token=" + vtok // TODO: replace with real app URL
	id, err := h.EmailSvc.Send(
		u.Email, u.Name,
		"Verify your LuxSuv account",
		"Click to verify: "+verifyURL,
		fmt.Sprintf(`<p>Hi %s,</p><p>Please <a href="%s">verify your email</a>. Link expires in 24 hours.</p>`, u.Name, verifyURL),
	)
	if err != nil {
		// In dev, don’t fail registration; just log why email didn’t go.
		// You can also include the URL in the JSON to make UX easy during development.
		fmt.Println("[email] send failed:", err)
		fmt.Println("[email] DEV verify URL:", verifyURL)
	} else {
		fmt.Println("[email] sent, id:", id)
	}

	// Do not sign in yet; require verification
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message":        "verification email sent",
		"dev_verify_url": verifyURL, // helpful while mail isn’t configured
	})
}

func (h *AuthHandler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	userID, err := h.Verify.ConsumeEmailVerification(r.Context(), token)
	if err != nil || userID == 0 {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	if err := h.Verify.MarkUserVerified(r.Context(), userID); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "email verified"})
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Email == "" || in.Password == "" {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	u, err := h.Users.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Block login if not verified
	verified, err := h.Verify.IsUserVerified(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !verified {
		http.Error(w, "email not verified", http.StatusUnauthorized)
		return
	}

	ok, _ := argon2id.ComparePasswordAndHash(in.Password, u.PasswordHash)
	if !ok {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Relink any old bookings
	_ = h.Users.LinkExistingBookings(r.Context(), u.ID, email)

	// Issue short-lived access token
	access, _ := auth.NewAccessToken(u.ID, u.Email, "rider", "bookings.read:self,bookings.write:self", 15*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": access,
		"user": map[string]any{
			"id": u.ID, "email": u.Email, "name": u.Name, "phone": u.Phone, "role": u.Role, "is_verified": true,
		},
	})
}
