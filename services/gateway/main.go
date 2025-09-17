package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	mw "github.com/diagnosis/luxsuv-bookings/pkg/middleware"
	"github.com/diagnosis/luxsuv-bookings/services/gateway/internal/handlers"
	"github.com/diagnosis/luxsuv-bookings/services/gateway/internal/proxy"
)

func main() {
	cfg := config.Load()
	
	// Initialize service proxies - use localhost for development, service names for production
	var (
		authBaseURL     = getServiceURL("AUTH_SERVICE_URL", "http://localhost:8081")
		bookingsBaseURL = getServiceURL("BOOKINGS_SERVICE_URL", "http://localhost:8082")
		dispatchBaseURL = getServiceURL("DISPATCH_SERVICE_URL", "http://localhost:8083")
		driverBaseURL   = getServiceURL("DRIVER_SERVICE_URL", "http://localhost:8084")
		paymentsBaseURL = getServiceURL("PAYMENTS_SERVICE_URL", "http://localhost:8085")
	)
	
	authProxy := proxy.NewServiceProxy(authBaseURL)
	bookingsProxy := proxy.NewServiceProxy(bookingsBaseURL)
	dispatchProxy := proxy.NewServiceProxy(dispatchBaseURL)
	driverProxy := proxy.NewServiceProxy(driverBaseURL)
	paymentsProxy := proxy.NewServiceProxy(paymentsBaseURL)
	
	// Initialize handlers
	h := handlers.New(authProxy, bookingsProxy, dispatchProxy, driverProxy, paymentsProxy, cfg)
	
	// Setup router
	r := chi.NewRouter()
	
	// Global middleware
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("gateway"))
	r.Use(mw.Logging)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.ErrorContext(r.Context(), "Panic recovered", "error", err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	})
	
	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:          300,
	}))
	
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// API routes
	r.Route("/v1", func(r chi.Router) {
		// Guest routes (no auth required for creation, session required for listing)
		r.Route("/guest", func(r chi.Router) {
			// Guest access routes (moved to auth service)
			r.Route("/access", func(r chi.Router) {
				r.Post("/request", h.GuestAccessRequest)
				r.Post("/verify", h.GuestAccessVerify)
				r.Post("/magic", h.GuestAccessMagic)
			})
			
			// Guest booking routes (routed to bookings service)
			r.Route("/bookings", func(r chi.Router) {
				r.Post("/", h.CreateGuestBooking)
				r.With(h.RequireGuestSession).Get("/", h.ListGuestBookings)
				r.With(h.OptionalGuestSession).Get("/{id}", h.GetGuestBooking)
				r.With(h.OptionalGuestSession).Patch("/{id}", h.UpdateGuestBooking)
				r.With(h.OptionalGuestSession).Delete("/{id}", h.CancelGuestBooking)
			})
		})
		
		// Auth routes (routed to auth service)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)
			r.Post("/verify-email", h.VerifyEmail)
			r.Post("/resend-verification", h.ResendVerification)
			r.Post("/refresh", h.RefreshToken)
		})
		
		// Admin user management routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(h.RequireJWT("admin"))
			r.Route("/users", func(r chi.Router) {
				r.Get("/", h.ListUsers)
				r.Get("/{id}", h.GetUser)
				r.Patch("/{id}", h.UpdateUser)
				r.Delete("/{id}", h.DeleteUser)
				r.Post("/{id}/roles", h.UpdateUserRole)
			})
		})
		
		// Rider routes (JWT required)
		r.Route("/rider", func(r chi.Router) {
			r.Use(h.RequireJWT("rider"))
			r.Route("/bookings", func(r chi.Router) {
				r.Post("/", h.CreateRiderBooking)
				r.Get("/", h.ListRiderBookings)
				r.Get("/{id}", h.GetRiderBooking)
				r.Patch("/{id}", h.UpdateRiderBooking)
				r.Delete("/{id}", h.CancelRiderBooking)
			})
		})
		
		// Driver routes (JWT required)
		r.Route("/driver", func(r chi.Router) {
			r.Use(h.RequireJWT("driver"))
			r.Get("/assignments", h.ListDriverAssignments)
			r.Post("/assignments/{id}/accept", h.AcceptAssignment)
			r.Post("/assignments/{id}/decline", h.DeclineAssignment)
			r.Post("/trips/{id}/start", h.StartTrip)
			r.Post("/trips/{id}/complete", h.CompleteTrip)
			r.Get("/availability", h.GetDriverAvailability)
			r.Post("/availability", h.SetDriverAvailability)
		})
		
		// Dispatch routes (JWT required, dispatcher role)
		r.Route("/dispatch", func(r chi.Router) {
			r.Use(h.RequireJWT("dispatcher"))
			r.Get("/pending", h.ListPendingBookings)
			r.Post("/assign", h.AssignBooking)
			r.Post("/reassign", h.ReassignBooking)
			r.Get("/drivers", h.ListAvailableDrivers)
		})
		
		// Admin booking routes (JWT required, admin role)
		r.Route("/admin", func(r chi.Router) {
			r.Use(h.RequireJWT("admin"))
			r.Route("/bookings", func(r chi.Router) {
				r.Get("/", h.ListAllBookings)
				r.Get("/{id}", h.GetBooking)
				r.Patch("/{id}", h.UpdateBooking)
				r.Delete("/{id}", h.CancelBooking)
			})
		})
		
		// Payment routes
		r.Route("/payments", func(r chi.Router) {
			r.Post("/intent", h.CreatePaymentIntent)
			r.Post("/webhook", h.StripeWebhook)
		})
	})
	
	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	
	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		logger.Info("Shutting down gateway service...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Gateway shutdown error", "error", err)
		}
	}()
	
	logger.Info("Starting gateway service", "port", cfg.Server.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Gateway server error", "error", err)
		os.Exit(1)
	}
}

func getServiceURL(envKey, fallback string) string {
	if url := os.Getenv(envKey); url != "" {
		return url
	}
	return fallback
}