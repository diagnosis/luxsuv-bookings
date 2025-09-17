package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/diagnosis/luxsuv-bookings/pkg/auth"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/repository"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	authService   service.AuthService
	guestService  service.GuestService
	rateLimitRepo repository.RateLimitRepository
	config        *config.Config
}

func New(
	authService service.AuthService,
	guestService service.GuestService,
	rateLimitRepo repository.RateLimitRepository,
	config *config.Config,
) *Handlers {
	return &Handlers{
		authService:   authService,
		guestService:  guestService,
		rateLimitRepo: rateLimitRepo,
		config:        config,
	}
}

// Middleware for JWT authentication
func (h *Handlers) RequireJWT(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "Missing or invalid authorization header", "UNAUTHORIZED")
				return
			}
			
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.Parse(token, h.config.Auth.JWTSecret)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "Invalid token", "INVALID_TOKEN")
				return
			}
			
			if requiredRole != "" && claims.Role != requiredRole && claims.Role != "admin" {
				writeError(w, http.StatusForbidden, "Insufficient permissions", "FORBIDDEN")
				return
			}
			
			// Add user context
			ctx := context.WithValue(r.Context(), logger.UserIDKey, claims.Sub)
			ctx = context.WithValue(ctx, "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Rate limiting middleware for guest access
func (h *Handlers) GuestAccessRateLimit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			key := "guest_access:" + clientIP
			
			allowed, err := h.rateLimitRepo.CheckRateLimit(r.Context(), key, 5, time.Minute)
			if err != nil {
				logger.ErrorContext(r.Context(), "Rate limit check failed", "error", err)
				// Allow request on error (fail open)
			} else if !allowed {
				writeError(w, http.StatusTooManyRequests, "Too many requests. Please try again later.", "RATE_LIMIT_EXCEEDED")
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions
func getClaims(r *http.Request) *auth.Claims {
	if claims, ok := r.Context().Value("claims").(*auth.Claims); ok {
		return claims
	}
	return nil
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	
	// Fall back to RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message, code string) {
	response := map[string]string{
		"error": message,
		"code":  code,
	}
	writeJSON(w, statusCode, response)
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0
	
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
	
	return limit, offset
}