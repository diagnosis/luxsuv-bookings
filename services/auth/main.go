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
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/handlers"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/repository"
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/service"
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
	userRepo := repository.NewUserRepository(pool)
	verifyRepo := repository.NewVerifyRepository(pool)
	
	// Initialize services
	authService := service.NewAuthService(userRepo, verifyRepo, eventBus, cfg)
	
	// Initialize handlers
	h := handlers.New(authService)
	
	// Setup router
	r := chi.NewRouter()
	
	// Middleware
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("auth"))
	r.Use(mw.Logging)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// Routes
	r.Route("/", func(r chi.Router) {
		// Guest access
		r.Post("/guest/access/request", h.GuestAccessRequest)
		r.Post("/guest/access/verify", h.GuestAccessVerify)
		r.Post("/guest/access/magic", h.GuestAccessMagic)
		
		// User auth
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/verify-email", h.VerifyEmail)
		r.Post("/resend-verification", h.ResendVerification)
		
		// Admin routes (require admin JWT)
		r.Route("/admin", func(r chi.Router) {
			r.Use(h.RequireJWT("admin"))
			r.Get("/users", h.ListUsers)
			r.Get("/users/{id}", h.GetUser)
			r.Patch("/users/{id}", h.UpdateUser)
			r.Delete("/users/{id}", h.DeleteUser)
		})
	})
	
	// Start server
	srv := &http.Server{
		Addr:         ":8081",
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
		
		logger.Info("Shutting down auth service...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Auth service shutdown error", "error", err)
		}
	}()
	
	logger.Info("Starting auth service", "port", "8081")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Auth service error", "error", err)
		os.Exit(1)
	}
}