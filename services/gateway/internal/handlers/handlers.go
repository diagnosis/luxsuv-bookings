package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	"github.com/diagnosis/luxsuv-bookings/services/gateway/internal/proxy"
)

type Handlers struct {
	authProxy     *proxy.ServiceProxy
	bookingsProxy *proxy.ServiceProxy
	dispatchProxy *proxy.ServiceProxy
	driverProxy   *proxy.ServiceProxy
	paymentsProxy *proxy.ServiceProxy
	config        *config.Config
}

func New(authProxy, bookingsProxy, dispatchProxy, driverProxy, paymentsProxy *proxy.ServiceProxy, config *config.Config) *Handlers {
	return &Handlers{
		authProxy:     authProxy,
		bookingsProxy: bookingsProxy,
		dispatchProxy: dispatchProxy,
		driverProxy:   driverProxy,
		paymentsProxy: paymentsProxy,
		config:        config,
	}
}

// Helper to proxy requests between gateway and services
func (h *Handlers) proxyRequest(w http.ResponseWriter, r *http.Request, serviceProxy *proxy.ServiceProxy, path string) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body", "INVALID_REQUEST")
		return
	}
	defer r.Body.Close()
	
	// Copy relevant headers
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 && shouldCopyHeader(key) {
			headers[key] = values[0]
		}
	}
	
	// Add query parameters to path
	if r.URL.RawQuery != "" {
		if strings.Contains(path, "?") {
			path += "&" + r.URL.RawQuery
		} else {
			path += "?" + r.URL.RawQuery
		}
	}
	
	// Make request to service
	resp, err := serviceProxy.ProxyRequest(r.Context(), r.Method, path, body, headers)
	if err != nil {
		logger.ErrorContext(r.Context(), "Service proxy error", "error", err, "path", path)
		writeError(w, http.StatusServiceUnavailable, "Service temporarily unavailable", "SERVICE_UNAVAILABLE")
		return
	}
	defer resp.Body.Close()
	
	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	
	// Copy status code
	w.WriteHeader(resp.StatusCode)
	
	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.ErrorContext(r.Context(), "Failed to copy response body", "error", err)
	}
}

// proxyAuthRequest specifically handles auth service routing
func (h *Handlers) proxyAuthRequest(w http.ResponseWriter, r *http.Request, path string) {
	h.proxyRequest(w, r, h.authProxy, path)
}

// proxyBookingsRequest specifically handles bookings service routing
func (h *Handlers) proxyBookingsRequest(w http.ResponseWriter, r *http.Request, path string) {
	h.proxyRequest(w, r, h.bookingsProxy, path)
}

func shouldCopyHeader(key string) bool {
	key = strings.ToLower(key)
	skipHeaders := []string{
		"host",
		"connection",
		"upgrade",
		"proxy-connection",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailers",
		"transfer-encoding",
	}
	
	for _, skip := range skipHeaders {
		if key == skip {
			return false
		}
	}
	return true
}

// JWT authentication middleware (validates tokens via auth service)
func (h *Handlers) RequireJWT(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "Missing or invalid authorization header", "UNAUTHORIZED")
				return
			}
			
			// Validate token via auth service
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			
			validateResp, err := h.authProxy.Post(ctx, "/validate-token", []byte(`{"token":"`+strings.TrimPrefix(authHeader, "Bearer ")+`","required_role":"`+requiredRole+`"}`), map[string]string{
				"Content-Type": "application/json",
			})
			if err != nil {
				logger.ErrorContext(r.Context(), "Token validation failed", "error", err)
				writeError(w, http.StatusUnauthorized, "Authentication service unavailable", "SERVICE_UNAVAILABLE")
				return
			}
			defer validateResp.Body.Close()
			
			if validateResp.StatusCode != http.StatusOK {
				writeError(w, validateResp.StatusCode, "Authentication failed", "UNAUTHORIZED")
				return
			}
			
			// Parse validation response to get user context
			var validationResult struct {
				UserID int64  `json:"user_id"`
				Email  string `json:"email"`
				Role   string `json:"role"`
			}
			if err := json.NewDecoder(validateResp.Body).Decode(&validationResult); err == nil {
				ctx = context.WithValue(r.Context(), logger.UserIDKey, validationResult.UserID)
				ctx = context.WithValue(ctx, "user_email", validationResult.Email)
				ctx = context.WithValue(ctx, "user_role", validationResult.Role)
				r = r.WithContext(ctx)
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// Guest session middleware (validates guest tokens via auth service)
func (h *Handlers) RequireGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			token = r.URL.Query().Get("session_token")
		}
		
		if token == "" {
			writeError(w, http.StatusUnauthorized, "Guest session required", "UNAUTHORIZED")
			return
		}
		
		// Validate guest token via auth service
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		
		validateResp, err := h.authProxy.Post(ctx, "/validate-guest-token", []byte(`{"token":"`+token+`"}`), map[string]string{
			"Content-Type": "application/json",
		})
		if err != nil {
			logger.ErrorContext(r.Context(), "Guest token validation failed", "error", err)
			writeError(w, http.StatusUnauthorized, "Authentication service unavailable", "SERVICE_UNAVAILABLE")
			return
		}
		defer validateResp.Body.Close()
		
		if validateResp.StatusCode != http.StatusOK {
			writeError(w, http.StatusUnauthorized, "Invalid guest session", "UNAUTHORIZED")
			return
		}
		
		// Add guest context
		var guestInfo struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(validateResp.Body).Decode(&guestInfo); err == nil {
			ctx = context.WithValue(r.Context(), "guest_email", guestInfo.Email)
			r = r.WithContext(ctx)
		}
		
		next.ServeHTTP(w, r)
	})
}

func (h *Handlers) OptionalGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for guest session token but don't require it
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			token = r.URL.Query().Get("session_token")
		}
		
		if token != "" {
			// Validate guest token via auth service
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			
			validateResp, err := h.authProxy.Post(ctx, "/validate-guest-token", []byte(`{"token":"`+token+`"}`), map[string]string{
				"Content-Type": "application/json",
			})
			if err == nil && validateResp.StatusCode == http.StatusOK {
				var guestInfo struct {
					Email string `json:"email"`
				}
				if err := json.NewDecoder(validateResp.Body).Decode(&guestInfo); err == nil {
					ctx = context.WithValue(r.Context(), "guest_email", guestInfo.Email)
					r = r.WithContext(ctx)
				}
				validateResp.Body.Close()
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// Helper functions
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

// Auth Service Routes
func (h *Handlers) GuestAccessRequest(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/guest/access/request")
}

func (h *Handlers) GuestAccessVerify(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/guest/access/verify")
}

func (h *Handlers) GuestAccessMagic(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/guest/access/magic")
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/register")
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/login")
}

func (h *Handlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/verify-email")
}

func (h *Handlers) ResendVerification(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/resend-verification")
}

func (h *Handlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/refresh")
}

// Admin User Management Routes
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, "/admin/users")
}

func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, r.URL.Path)
}

func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, r.URL.Path)
}

func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, r.URL.Path)
}

func (h *Handlers) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	h.proxyAuthRequest(w, r, r.URL.Path)
}

// Bookings Service Routes
func (h *Handlers) CreateGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, "/guest/bookings")
}

func (h *Handlers) ListGuestBookings(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, "/guest/bookings")
}

func (h *Handlers) GetGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

func (h *Handlers) UpdateGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

func (h *Handlers) CancelGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

func (h *Handlers) CreateRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, "/rider/bookings")
}

func (h *Handlers) ListRiderBookings(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, "/rider/bookings")
}

func (h *Handlers) GetRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

func (h *Handlers) UpdateRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

func (h *Handlers) CancelRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

// Admin Booking Routes
func (h *Handlers) ListAllBookings(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, "/admin/bookings")
}

func (h *Handlers) GetBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

func (h *Handlers) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

func (h *Handlers) CancelBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyBookingsRequest(w, r, r.URL.Path)
}

// Driver Routes (TODO: Implement in driver service)
func (h *Handlers) ListDriverAssignments(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, "/assignments")
}

func (h *Handlers) AcceptAssignment(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, r.URL.Path)
}

func (h *Handlers) DeclineAssignment(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, r.URL.Path)
}

func (h *Handlers) StartTrip(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, r.URL.Path)
}

func (h *Handlers) CompleteTrip(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, r.URL.Path)
}

func (h *Handlers) GetDriverAvailability(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, "/availability")
}

func (h *Handlers) SetDriverAvailability(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, "/availability")
}

// Dispatch Routes (TODO: Implement in dispatch service)
func (h *Handlers) ListPendingBookings(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.dispatchProxy, "/pending")
}

func (h *Handlers) AssignBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.dispatchProxy, "/assign")
}

func (h *Handlers) ReassignBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.dispatchProxy, "/reassign")
}

func (h *Handlers) ListAvailableDrivers(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.dispatchProxy, "/drivers")
}

// Payment Routes (TODO: Implement in payments service)
func (h *Handlers) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.paymentsProxy, "/intent")
}

func (h *Handlers) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.paymentsProxy, "/webhook")
}