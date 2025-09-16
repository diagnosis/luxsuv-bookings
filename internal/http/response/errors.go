package response

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorResponse represents a structured JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// WriteError writes a structured JSON error response
func WriteError(w http.ResponseWriter, statusCode int, message string, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errResp := ErrorResponse{
		Error: message,
		Code:  code,
	}
	
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

// WriteErrorWithDetails writes a structured JSON error response with additional details
func WriteErrorWithDetails(w http.ResponseWriter, statusCode int, message, code, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errResp := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}
	
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

// Common error codes
const (
	CodeInvalidInput     = "INVALID_INPUT"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeRateLimit        = "RATE_LIMIT_EXCEEDED"
	CodeInternalError    = "INTERNAL_ERROR"
	CodeExpiredToken     = "EXPIRED_TOKEN"
	CodeInvalidToken     = "INVALID_TOKEN"
	CodePastDateTime     = "PAST_DATETIME"
	CodeEmailExists      = "EMAIL_EXISTS"
	CodeBookingCanceled  = "BOOKING_CANCELED"
)

// Convenience functions for common errors
func BadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, message, CodeInvalidInput)
}

func Unauthorized(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnauthorized, message, CodeUnauthorized)
}

func Forbidden(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, message, CodeForbidden)
}

func NotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, message, CodeNotFound)
}

func InternalError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, message, CodeInternalError)
}

func RateLimit(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusTooManyRequests, message, CodeRateLimit)
}

func Conflict(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusConflict, message, CodeConflict)
}