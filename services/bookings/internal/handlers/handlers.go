package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/diagnosis/luxsuv-bookings/pkg/auth"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	bookingService service.BookingService
	guestService   service.GuestService
	config         *config.Config
}

func New(bookingService service.BookingService, guestService service.GuestService) *Handlers {
	return &Handlers{
		bookingService: bookingService,
		guestService:   guestService,
		config:         config.Load(),
	}
}

// Middleware for JWT authentication
func (h *Handlers) RequireJWT(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
				return
			}
			
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.Parse(token, h.config.Auth.JWTSecret)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			
			if requiredRole != "" && claims.Role != requiredRole && claims.Role != "admin" {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}
			
			// Add user context
			ctx := context.WithValue(r.Context(), logger.UserIDKey, claims.Sub)
			ctx = context.WithValue(ctx, "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Middleware for guest session authentication
func (h *Handlers) RequireGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			token = r.URL.Query().Get("session_token")
		}
		
		if token == "" {
			http.Error(w, "Guest session required", http.StatusUnauthorized)
			return
		}
		
		claims, err := auth.Parse(token, h.config.Auth.JWTSecret)
		if err != nil || claims.Role != "guest" {
			http.Error(w, "Invalid guest session", http.StatusUnauthorized)
			return
		}
		
		ctx := context.WithValue(r.Context(), "guest_claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Middleware for optional guest session
func (h *Handlers) OptionalGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			token = r.URL.Query().Get("session_token")
		}
		
		if token != "" {
			if claims, err := auth.Parse(token, h.config.Auth.JWTSecret); err == nil && claims.Role == "guest" {
				ctx := context.WithValue(r.Context(), "guest_claims", claims)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Helper to get user claims from context
func getClaims(r *http.Request) *auth.Claims {
	if claims, ok := r.Context().Value("claims").(*auth.Claims); ok {
		return claims
	}
	return nil
}

// Helper to get guest claims from context
func getGuestClaims(r *http.Request) *auth.Claims {
	if claims, ok := r.Context().Value("guest_claims").(*auth.Claims); ok {
		return claims
	}
	return nil
}

// Helper functions for common response patterns
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

// Helper to parse pagination parameters
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