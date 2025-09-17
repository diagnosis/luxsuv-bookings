package handlers

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/domain"
)

// GuestAccessRequest handles guest access code requests
func (h *Handlers) GuestAccessRequest(w http.ResponseWriter, r *http.Request) {
	var req domain.GuestAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_INPUT")
		return
	}
	
	clientIP := getClientIP(r)
	clientIPNet := net.ParseIP(clientIP)
	if clientIPNet == nil {
		clientIPNet = net.ParseIP("0.0.0.0") // Fallback
	}
	
	if err := h.guestService.RequestAccess(r.Context(), &req, clientIPNet); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "ACCESS_REQUEST_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Access code sent to your email",
	})
}

// GuestAccessVerify handles guest access code verification
func (h *Handlers) GuestAccessVerify(w http.ResponseWriter, r *http.Request) {
	var req domain.GuestAccessVerify
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON format", "INVALID_INPUT")
		return
	}
	
	response, err := h.guestService.VerifyCode(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error(), "VERIFICATION_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, response)
}

// GuestAccessMagic handles magic link verification
func (h *Handlers) GuestAccessMagic(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "Missing magic token", "INVALID_INPUT")
		return
	}
	
	response, err := h.guestService.VerifyMagicLink(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error(), "MAGIC_LINK_FAILED")
		return
	}
	
	writeJSON(w, http.StatusOK, response)
}