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
	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/mailer"
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

	// Initialize mailer
	var emailSvc mailer.Service
	if cfg.Email.SMTPHost != "" {
		emailSvc = mailer.NewSMTPMailer(
			cfg.Email.SMTPHost,
			cfg.Email.SMTPPort,
			cfg.Email.SMTPFrom,
			cfg.Email.SMTPUser,
			cfg.Email.SMTPPass,
			cfg.Email.SMTPUseTLS,
		)
		logger.Info("Mailer initialized", "type", "SMTP", "host", cfg.Email.SMTPHost)
	} else if cfg.Email.MailerSendKey != "" {
		emailSvc = mailer.NewMailerSend(
			cfg.Email.MailerSendKey,
			"LuxSUV Support",
			cfg.Email.SMTPFrom,
		)
		logger.Info("Mailer initialized", "type", "MailerSend")
	} else {
		emailSvc = mailer.NewDevMailer()
		logger.Info("Mailer initialized", "type", "Development")
	}
	
	// Initialize repositories
	userRepo := repository.NewUserRepository(pool)
	verifyRepo := repository.NewVerifyRepository(pool)
	rateLimitRepo := repository.NewRateLimitRepository(pool)
	
	// Initialize services
	authService := service.NewAuthService(userRepo, verifyRepo, emailSvc, eventBus, cfg)
	guestService := service.NewGuestService(verifyRepo, userRepo, emailSvc, eventBus, cfg)
	
	// Initialize handlers
	h := handlers.New(authService, guestService, rateLimitRepo, cfg)
	
	// Setup router
	r := chi.NewRouter()
	
	// Middleware
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("auth"))
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// Routes
	r.Route("/", func(r chi.Router) {
		// Guest access routes (with rate limiting)
		r.Route("/guest/access", func(r chi.Router) {
			r.Use(h.GuestAccessRateLimit())
			r.Post("/request", h.GuestAccessRequest)
			r.Post("/verify", h.GuestAccessVerify)
			r.Post("/magic", h.GuestAccessMagic)
		})
		
		// User authentication routes
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/verify-email", h.VerifyEmail)
		r.Post("/resend-verification", h.ResendVerification)
		r.Post("/refresh", h.RefreshToken)
		
		// Admin routes (require admin JWT)
		r.Route("/admin", func(r chi.Router) {
			r.Use(h.RequireJWT("admin"))
			r.Get("/users", h.ListUsers)
			r.Get("/users/{id}", h.GetUser)
			r.Patch("/users/{id}", h.UpdateUser)
			r.Delete("/users/{id}", h.DeleteUser)
			r.Post("/users/{id}/roles", h.UpdateUserRole)
		})
	})
	
	// Background cleanup task
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if deleted, err := verifyRepo.DeleteExpiredTokens(context.Background()); err != nil {
				logger.Error("Failed to cleanup expired verification tokens", "error", err)
			} else if deleted > 0 {
				logger.Info("Cleaned up expired verification tokens", "count", deleted)
			}
			
			if deleted, err := rateLimitRepo.CleanupExpired(context.Background()); err != nil {
				logger.Error("Failed to cleanup expired rate limits", "error", err)
			} else if deleted > 0 {
				logger.Info("Cleaned up expired rate limits", "count", deleted)
			}
		}
	}()
	
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