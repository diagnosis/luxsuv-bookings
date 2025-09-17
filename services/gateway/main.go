package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	mw "github.com/diagnosis/luxsuv-bookings/pkg/middleware"
	"github.com/diagnosis/luxsuv-bookings/services/gateway/internal/handlers"
	"github.com/diagnosis/luxsuv-bookings/services/gateway/internal/proxy"
)

func main() {
	cfg := config.Load()
	
	// Initialize service proxies
	authProxy := proxy.NewServiceProxy("http://auth-svc:8081")
	bookingsProxy := proxy.NewServiceProxy("http://bookings-svc:8082")
	dispatchProxy := proxy.NewServiceProxy("http://dispatch-svc:8083")
	driverProxy := proxy.NewServiceProxy("http://driver-svc:8084")
	paymentsProxy := proxy.NewServiceProxy("http://payments-svc:8085")
	
	// Initialize handlers
	h := handlers.New(authProxy, bookingsProxy, dispatchProxy, driverProxy, paymentsProxy)
	
	// Setup router
	r := chi.NewRouter()
	
	// Global middleware
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("gateway"))
	r.Use(mw.Logging)
	r.Use(middleware.Recoverer)
	r.Use(mw.CORS)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// API routes
	r.Route("/v1", func(r chi.Router) {
		// Guest routes (no auth required for creation, session required for listing)
		r.Route("/guest", func(r chi.Router) {
			r.Route("/access", func(r chi.Router) {
				r.Post("/request", h.GuestAccessRequest)
				r.Post("/verify", h.GuestAccessVerify)
				r.Post("/magic", h.GuestAccessMagic)
			})
			r.Route("/bookings", func(r chi.Router) {
				r.Post("/", h.CreateGuestBooking)
				r.With(h.RequireGuestSession).Get("/", h.ListGuestBookings)
				r.With(h.OptionalGuestSession).Get("/{id}", h.GetGuestBooking)
				r.With(h.OptionalGuestSession).Patch("/{id}", h.UpdateGuestBooking)
				r.With(h.OptionalGuestSession).Delete("/{id}", h.CancelGuestBooking)
			})
		})
		
		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)
			r.Post("/verify-email", h.VerifyEmail)
			r.Post("/resend-verification", h.ResendVerification)
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
		
		// Admin routes (JWT required, admin role)
		r.Route("/admin", func(r chi.Router) {
			r.Use(h.RequireJWT("admin"))
			r.Route("/users", func(r chi.Router) {
				r.Get("/", h.ListUsers)
				r.Get("/{id}", h.GetUser)
				r.Patch("/{id}", h.UpdateUser)
				r.Delete("/{id}", h.DeleteUser)
			})
			r.Route("/bookings", func(r chi.Router) {
				r.Get("/", h.ListAllBookings)
				r.Get("/{id}", h.GetBooking)
				r.Patch("/{id}", h.UpdateBooking)
				r.Delete("/{id}", h.CancelBooking)
			})
			r.Route("/drivers", func(r chi.Router) {
				r.Get("/", h.ListAllDrivers)
				r.Post("/", h.CreateDriver)
				r.Patch("/{id}", h.UpdateDriver)
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