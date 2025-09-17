package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/domain"
	"github.com/go-chi/chi/v5"
)

// Register handles user registration
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_INPUT")
		return
	}
	
	user, verifyURL, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "REGISTRATION_FAILED")
		return
	}
	
	response := map[string]interface{}{
		"message": "Registration successful. Please check your email to verify your account.",
		"user":    user.ToUserInfo(),
	}
	
	// Include verify URL in development mode
	if h.config.Email.DevMode {
		response["dev_verify_url"] = verifyURL
	}
	
	writeJSON(w, http.StatusCreated, response)
}

// Login handles user authentication
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_INPUT")
		return
	}
	
	response, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error(), "LOGIN_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, response)
}

// VerifyEmail handles email verification
func (h *Handlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "Missing verification token", "INVALID_INPUT")
		return
	}
	
	user, err := h.authService.VerifyEmail(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "VERIFICATION_FAILED")
		return
	}
	
	response := map[string]interface{}{
		"message": "Email verified successfully",
		"user":    user.ToUserInfo(),
	}
	
	writeJSON(w, http.StatusOK, response)
}

// ResendVerification handles resending verification emails
func (h *Handlers) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		writeError(w, http.StatusBadRequest, "Email is required", "INVALID_INPUT")
		return
	}
	
	if err := h.authService.ResendVerification(r.Context(), req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "RESEND_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent",
	})
}

// RefreshToken handles token refresh
func (h *Handlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_INPUT")
		return
	}
	
	response, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error(), "REFRESH_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, response)
}

// Admin handlers

// ListUsers handles listing all users (admin only)
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	
	users, err := h.authService.ListUsers(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list users", "INTERNAL_ERROR")
		return
	}
	
	// Convert to user info (without sensitive data)
	userInfos := make([]*domain.UserInfo, len(users))
	for i, user := range users {
		userInfos[i] = user.ToUserInfo()
	}
	
	writeJSON(w, http.StatusOK, userInfos)
}

// GetUser handles getting a specific user (admin only)
func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID", "INVALID_INPUT")
		return
	}
	
	user, err := h.authService.GetUser(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found", "NOT_FOUND")
		return
	}
	
	writeJSON(w, http.StatusOK, user.ToUserInfo())
}

// UpdateUser handles updating user information (admin only)
func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID", "INVALID_INPUT")
		return
	}
	
	var req domain.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_INPUT")
		return
	}
	
	user, err := h.authService.UpdateUser(r.Context(), id, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "UPDATE_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, user.ToUserInfo())
}

// DeleteUser handles deleting a user (admin only)
func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID", "INVALID_INPUT")
		return
	}
	
	if err := h.authService.DeleteUser(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "DELETE_FAILED")
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// UpdateUserRole handles updating user roles (admin only)
func (h *Handlers) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid user ID", "INVALID_INPUT")
		return
	}
	
	var req domain.UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_INPUT")
		return
	}
	
	if err := h.authService.UpdateUserRole(r.Context(), id, req.Role); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "ROLE_UPDATE_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "User role updated successfully",
	})
}