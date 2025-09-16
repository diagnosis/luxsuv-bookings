package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/database"
	"github.com/diagnosis/luxsuv-bookings/internal/http/handlers"
	"github.com/diagnosis/luxsuv-bookings/internal/http/handlers/guest"
	mw "github.com/diagnosis/luxsuv-bookings/internal/http/middleware"
	"github.com/diagnosis/luxsuv-bookings/internal/platform/mailer"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()
	pool, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	log.Println("connected to database")
	var emailSvc mailer.Service
	if os.Getenv("SMTP_HOST") != "" {
		host := os.Getenv("SMTP_HOST") // "localhost"
		port := 1025
		if v := os.Getenv("SMTP_PORT"); v != "" {
			if p, err := strconv.Atoi(v); err == nil && p > 0 {
				port = p
			}
		}
		from := os.Getenv("SMTP_FROM") // "dev@luxsuv.local"
		user := os.Getenv("SMTP_USER") // leave empty for Mailpit
		pass := os.Getenv("SMTP_PASS") // leave empty for Mailpit
		useTLS := os.Getenv("SMTP_USE_TLS") == "1"

		emailSvc = mailer.NewSMTPMailer(host, port, from, user, pass, useTLS)
		log.Printf("mailer: SMTP mode host=%s port=%d from=%s", host, port, from)
	} else {
		// keep your MailerSend path available for later/staging
		emailSvc = mailer.NewMailer(
			os.Getenv("MAILERSEND_API_KEY"),
			"LuxSuv Support",
			os.Getenv("MAILER_FROM"),
		)
	}

	//repo and handlers
	bookRepo := postgres.NewBookingRepo(pool)
	idempotencyRepo := postgres.NewIdempotencyRepo(pool)
	userRepo := postgres.NewUsersRepo(pool)
	verifyRepo := postgres.NewVerifyRepo(pool)
	//
	guestBookings := guest.NewBookingsHandler(bookRepo, idempotencyRepo)
	guestAccess := guest.NewAccessHandler(verifyRepo, emailSvc)

	// Rate limiting for guest access requests
	accessRateLimit := mw.NewRateLimiter(pool, mw.RateLimitConfig{
		Requests: 5,                           // 5 requests per window
		Window:   time.Minute,                 // 1 minute window
		KeyFunc:  mw.GuestAccessRateLimitKeyFunc,
	})

	authH := handlers.NewAuthHandler(userRepo, verifyRepo, emailSvc)
	riderH := handlers.NewRiderBookingsHandler(bookRepo, userRepo)

	//router
	r := chi.NewRouter()
	//add mws
	r.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		middleware.RedirectSlashes,
		cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
			AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}, // add PATCH
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}),
	)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})
	//
	r.Mount("/v1/guest/bookings", guestBookings.Routes())

	// Mount guest access with rate limiting
	r.Group(func(gr chi.Router) {
		gr.Use(accessRateLimit.Middleware())
		gr.Mount("/v1/guest/access", guestAccess.Routes())
	})

	r.Mount("/v1/auth", authH.Routes())
	r.Group(func(gr chi.Router) {
		gr.Use(mw.RequireJWT)
		gr.Mount("/v1/rider/bookings", riderH.Routes())
	})
	//r.Mount("/v1/rider/bookings", riderH.Routes())

	addr := ":" + env("PORT", "8080")
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Println("starting server on " + addr)
	log.Fatal(srv.ListenAndServe())
}
func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
