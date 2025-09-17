package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/diagnosis/luxsuv-bookings/pkg/config"
	"github.com/diagnosis/luxsuv-bookings/pkg/database"
	"github.com/diagnosis/luxsuv-bookings/pkg/events"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	mw "github.com/diagnosis/luxsuv-bookings/pkg/middleware"
	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/handlers"
	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/repository"
	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/service"
)

func main() {
	cfg := config.Load()
	
	// Connect to database
	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.Database.URL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	
	// Connect to event bus
	eventBus, err := events.NewNATSEventBus(cfg.NATS.URL)
	if err != nil {
		logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer eventBus.Close()
	
	// Initialize repositories
	bookingRepo := repository.NewBookingRepository(pool)
	idempotencyRepo := repository.NewIdempotencyRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	verifyRepo := repository.NewVerifyRepository(pool)
	
	// Initialize services
	bookingService := service.NewBookingService(bookingRepo, idempotencyRepo, userRepo, eventBus, cfg)
	guestService := service.NewGuestService(verifyRepo, userRepo, eventBus, cfg)
	
	// Initialize handlers
	h := handlers.New(bookingService, guestService)
	
	// Setup router
	r := chi.NewRouter()
	
	// Middleware
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("bookings"))
	r.Use(mw.Logging)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// Routes - maintain existing API structure
	r.Route("/", func(r chi.Router) {
		// Guest booking routes
		r.Route("/guest/bookings", func(r chi.Router) {
			r.Post("/", h.CreateGuestBooking)
			r.With(h.RequireGuestSession).Get("/", h.ListGuestBookings)
			r.With(h.OptionalGuestSession).Get("/{id}", h.GetGuestBooking)
			r.With(h.OptionalGuestSession).Patch("/{id}", h.UpdateGuestBooking)
			r.With(h.OptionalGuestSession).Delete("/{id}", h.CancelGuestBooking)
		})
		
		// Rider booking routes (JWT required)
		r.Route("/rider/bookings", func(r chi.Router) {
			r.Use(h.RequireJWT("rider"))
			r.Post("/", h.CreateRiderBooking)
			r.Get("/", h.ListRiderBookings)
			r.Get("/{id}", h.GetRiderBooking)
			r.Patch("/{id}", h.UpdateRiderBooking)
			r.Delete("/{id}", h.CancelRiderBooking)
		})
		
		// Admin booking routes
		r.Route("/admin/bookings", func(r chi.Router) {
			r.Use(h.RequireJWT("admin"))
			r.Get("/", h.ListAllBookings)
			r.Get("/{id}", h.GetBooking)
			r.Patch("/{id}", h.UpdateBooking)
			r.Delete("/{id}", h.CancelBooking)
		})
	})
	
	// Start server
	srv := &http.Server{
		Addr:         ":8082",
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
		
		logger.Info("Shutting down bookings service...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Bookings service shutdown error", "error", err)
		}
	}()
	
	logger.Info("Starting bookings service", "port", "8082")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Bookings service error", "error", err)
		os.Exit(1)
	}
}