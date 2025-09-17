package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/diagnosis/luxsuv-bookings/pkg/auth"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	"github.com/diagnosis/luxsuv-bookings/services/gateway/internal/proxy"
)

type Handlers struct {
	authProxy     *proxy.ServiceProxy
	bookingsProxy *proxy.ServiceProxy
	dispatchProxy *proxy.ServiceProxy
	driverProxy   *proxy.ServiceProxy
	paymentsProxy *proxy.ServiceProxy
}

func New(authProxy, bookingsProxy, dispatchProxy, driverProxy, paymentsProxy *proxy.ServiceProxy) *Handlers {
	return &Handlers{
		authProxy:     authProxy,
		bookingsProxy: bookingsProxy,
		dispatchProxy: dispatchProxy,
		driverProxy:   driverProxy,
		paymentsProxy: paymentsProxy,
	}
}

// Helper to copy request body and headers
func (h *Handlers) proxyRequest(w http.ResponseWriter, r *http.Request, serviceProxy *proxy.ServiceProxy, path string) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
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
	
	// Make request to service
	resp, err := serviceProxy.ProxyRequest(r.Context(), r.Method, path, body, headers)
	if err != nil {
		logger.ErrorContext(r.Context(), "Service proxy error", "error", err, "path", path)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
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

// Auth middleware
func (h *Handlers) RequireJWT(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
				return
			}
			
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.Parse(token)
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
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *Handlers) RequireGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for session token in header or query param
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
		
		claims, err := auth.Parse(token)
		if err != nil || claims.Role != "guest" {
			http.Error(w, "Invalid guest session", http.StatusUnauthorized)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (h *Handlers) OptionalGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This middleware allows both token-based and session-based access
		// The actual service will handle the authorization logic
		next.ServeHTTP(w, r)
	})
}

// Guest Access Handlers
func (h *Handlers) GuestAccessRequest(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/guest/access/request")
}

func (h *Handlers) GuestAccessVerify(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/guest/access/verify")
}

func (h *Handlers) GuestAccessMagic(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/guest/access/magic")
}

// Guest Booking Handlers
func (h *Handlers) CreateGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, "/guest/bookings")
}

func (h *Handlers) ListGuestBookings(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, "/guest/bookings"+r.URL.RawQuery)
}

func (h *Handlers) GetGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path+r.URL.RawQuery)
}

func (h *Handlers) UpdateGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path+r.URL.RawQuery)
}

func (h *Handlers) CancelGuestBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path+r.URL.RawQuery)
}

// Auth Handlers
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/register")
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/login")
}

func (h *Handlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/verify-email")
}

func (h *Handlers) ResendVerification(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/resend-verification")
}

// Rider Booking Handlers
func (h *Handlers) CreateRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, "/rider/bookings")
}

func (h *Handlers) ListRiderBookings(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, "/rider/bookings"+r.URL.RawQuery)
}

func (h *Handlers) GetRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path)
}

func (h *Handlers) UpdateRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path)
}

func (h *Handlers) CancelRiderBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path)
}

// Driver Handlers
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

// Dispatch Handlers
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

// Admin Handlers
func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, "/admin/users")
}

func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, r.URL.Path)
}

func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, r.URL.Path)
}

func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.authProxy, r.URL.Path)
}

func (h *Handlers) ListAllBookings(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, "/admin/bookings")
}

func (h *Handlers) GetBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path)
}

func (h *Handlers) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path)
}

func (h *Handlers) CancelBooking(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.bookingsProxy, r.URL.Path)
}

func (h *Handlers) ListAllDrivers(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, "/admin/drivers")
}

func (h *Handlers) CreateDriver(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, "/admin/drivers")
}

func (h *Handlers) UpdateDriver(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.driverProxy, r.URL.Path)
}

// Payment Handlers
func (h *Handlers) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.paymentsProxy, "/intent")
}

func (h *Handlers) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, h.paymentsProxy, "/webhook")
}