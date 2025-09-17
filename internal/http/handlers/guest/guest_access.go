package guest

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/http/response"
	"github.com/diagnosis/luxsuv-bookings/internal/platform/auth"
	"github.com/diagnosis/luxsuv-bookings/internal/platform/mailer"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/diagnosis/luxsuv-bookings/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AccessHandler struct {
	Verify   postgres.VerifyRepo
	EmailSvc mailer.Service
	UsersRepo postgres.UsersRepo
}

func NewAccessHandler(verify postgres.VerifyRepo, emailSvc mailer.Service, usersRepo postgres.UsersRepo) *AccessHandler {
	return &AccessHandler{Verify: verify, EmailSvc: emailSvc, UsersRepo: usersRepo}
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
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "Invalid JSON format")
		return
	}

	// Normalize and validate email
	in.Email = utils.NormalizeEmail(in.Email)
	if in.Email == "" {
		response.WriteError(w, http.StatusBadRequest, "Email is required", response.CodeInvalidInput)
		return
	}

	if !utils.IsValidEmail(in.Email) {
		response.WriteError(w, http.StatusBadRequest, "Invalid email format", response.CodeInvalidInput)
		return
	}

	// Check if this email belongs to a registered user
	if user, err := h.UsersRepo.FindByEmail(r.Context(), in.Email); err == nil && user != nil {
		// Email belongs to a registered user - they should login instead
		response.WriteError(w, http.StatusForbidden, "This email is associated with a registered account. Please login with your password instead.", response.CodeForbidden)
		return
	}

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

	if err := h.Verify.CreateGuestAccess(r.Context(), in.Email, codeHash, magic, expires, ip); err != nil {
		log.Printf("failed to create guest access: %v", err)
		response.InternalError(w, "Failed to create access code")
		return
	}

	link := "http://localhost:5173/guest/access?token=" + magic
	if err := h.EmailSvc.SendGuestAccess(in.Email, code, link); err != nil {
		log.Printf("failed to send guest access email to %s: %v", in.Email, err)
		// Don't fail the request - code was created successfully
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Access code sent to your email",
	})
}

type verifyIn struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *AccessHandler) verify(w http.ResponseWriter, r *http.Request) {
	var in verifyIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "Invalid JSON format")
		return
	}

	// Normalize and validate inputs
	in.Email = utils.NormalizeEmail(in.Email)
	in.Code = strings.TrimSpace(in.Code)

	if in.Email == "" || in.Code == "" {
		response.WriteError(w, http.StatusBadRequest, "Email and code are required", response.CodeInvalidInput)
		return
	}

	if !utils.IsValidEmail(in.Email) {
		response.WriteError(w, http.StatusBadRequest, "Invalid email format", response.CodeInvalidInput)
		return
	}

	// Validate code format (should be 6 digits)
	if len(in.Code) != 6 {
		response.WriteError(w, http.StatusBadRequest, "Code must be 6 digits", response.CodeInvalidInput)
		return
	}

	// Check if this email belongs to a registered user
	if user, err := h.UsersRepo.FindByEmail(r.Context(), in.Email); err == nil && user != nil {
		// Email belongs to a registered user - they should login instead
		response.WriteError(w, http.StatusForbidden, "This email is associated with a registered account. Please login with your password instead.", response.CodeForbidden)
		return
	}

	ok, err := h.Verify.CheckGuestCode(r.Context(), in.Email, in.Code)
	if err != nil {
		log.Printf("failed to check guest code: %v", err)
		response.InternalError(w, "Failed to verify code")
		return
	}
	
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "Invalid or expired code", response.CodeExpiredToken)
		return
	}

	token, err := auth.NewGuestSession(in.Email, 30*time.Minute)
	if err != nil {
		log.Printf("failed to create guest session token: %v", err)
		response.InternalError(w, "Failed to create session")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"session_token": token, 
		"expires_in": int64(1800),
	})
}

func (h *AccessHandler) magic(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		response.WriteError(w, http.StatusBadRequest, "Token parameter is required", response.CodeInvalidInput)
		return
	}

	email, ok, err := h.Verify.ConsumeGuestMagic(r.Context(), token)
	if err != nil {
		log.Printf("failed to consume guest magic token: %v", err)
		response.InternalError(w, "Failed to process magic link")
		return
	}
	
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "Invalid or expired magic link", response.CodeExpiredToken)
		return
	}

	jwt, err := auth.NewGuestSession(email, 30*time.Minute)
	if err != nil {
		log.Printf("failed to create guest session from magic link: %v", err)
		response.InternalError(w, "Failed to create session")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"session_token": jwt, 
		"expires_in": int64(1800),
	})
}
