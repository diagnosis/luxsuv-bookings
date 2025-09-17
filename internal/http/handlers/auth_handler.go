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

// Add rate limiting for auth endpoints
func (h *AuthHandler) rateLimitedRoutes() chi.Router {
	// Create rate limiter for auth endpoints
	authRateLimit := mw.NewRateLimiter(h.pool, mw.RateLimitConfig{
		Requests: 10,              // 10 requests per window
		Window:   time.Minute,     // 1 minute window  
		KeyFunc: func(r *http.Request) []string {
			ip := mw.GetClientIP(r)
			return []string{"auth:" + ip}
		},
	})
	
	r := chi.NewRouter()
	r.Use(authRateLimit.Middleware())
	return r
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	
	// Add rate limiting to verification endpoint
	r.Group(func(rr chi.Router) {
		// Rate limit verification attempts
		verifyRateLimit := mw.NewRateLimiter(pool, mw.RateLimitConfig{
			Requests: 5,                    // 5 verification attempts per window
			Window:   5 * time.Minute,      // 5 minute window
			KeyFunc: func(r *http.Request) []string {
				token := r.URL.Query().Get("token")
				if token != "" {
					return []string{"verify:" + token}
				}
				ip := mw.GetClientIP(r)
				return []string{"verify:" + ip}
			},
		})
		rr.Use(verifyRateLimit.Middleware())
		rr.Post("/verify-email", h.verifyEmail)
	})
	
	// Add resend verification endpoint
	r.Post("/resend-verification", h.resendVerification)
	
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
	if err := h.Verify.CreateEmailVerification(r.Context(), u.ID, vtok, time.Now().Add(2*time.Hour)); err != nil {
		log.Printf("failed to create email verification token: %v", err)
		response.InternalError(w, "Failed to create verification token")
		return
	}

	// Get base URL from environment, fallback to localhost for development
	baseURL := os.Getenv("FRONTEND_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	verifyURL := baseURL + "/verify-email?token=" + vtok
	
	id, err := h.EmailSvc.Send(
		u.Email, u.Name,
		"Verify your LuxSuv account",
		"Click to verify: "+verifyURL,
		fmt.Sprintf(`<p>Hi %s,</p><p>Please <a href="%s">verify your email</a>. Link expires in 24 hours.</p>`, u.Name, verifyURL),
	)
	if err != nil {
		log.Printf("failed to send verification email to %s: %v", u.Email, err)
		// In production, you might want to fail registration if email can't be sent
		// For now, we'll continue but notify the user
		fmt.Println("[email] send failed:", err)
		fmt.Println("[email] DEV verify URL:", verifyURL)
	} else {
		fmt.Println("[email] sent, id:", id)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	response := map[string]string{
		"message":        "verification email sent",
	}
	
	// Include verify URL in development mode
	if os.Getenv("ENVIRONMENT") == "development" {
		response["dev_verify_url"] = verifyURL
	}
	
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		response.WriteError(w, http.StatusBadRequest, "Missing verification token", response.CodeInvalidInput)
		return
	}
	
	userID, err := h.Verify.ConsumeEmailVerification(r.Context(), token)
	if err != nil {
		log.Printf("failed to consume email verification token: %v", err)
		response.InternalError(w, "Failed to process verification")
		return
	}
	if userID == 0 {
		response.WriteError(w, http.StatusUnauthorized, "Invalid or expired verification token", response.CodeExpiredToken)
		return
	}
	
	if err := h.Verify.MarkUserVerified(r.Context(), userID); err != nil {
		log.Printf("failed to mark user as verified: %v", err)
		response.InternalError(w, "Failed to verify account")
		return
	}
	
	// Get user details for response
	u, err := h.Users.FindByID(r.Context(), userID)
	if err != nil {
		log.Printf("failed to get user after verification: %v", err)
		// Still return success since verification worked
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	responseData := map[string]interface{}{
		"message": "Email verified successfully",
		"verified": true,
	}
	
	// Optionally include user info
	if u != nil {
		responseData["user"] = map[string]interface{}{
			"id": u.ID,
			"email": u.Email,
			"name": u.Name,
		}
	}
	
	_ = json.NewEncoder(w).Encode(responseData)
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
