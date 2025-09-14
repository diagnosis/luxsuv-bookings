package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/database"
	"github.com/diagnosis/luxsuv-bookings/internal/http/handlers"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	ctx := context.Background()
	pool, err := database.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	log.Println("connected to database")

	//repo and handlers
	repo := postgres.NewBookingRepo(pool)
	gh := handlers.NewBookingGuestHandler(repo)

	//router
	r := chi.NewRouter()
	//add mws
	r.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Logger,
		middleware.Recoverer,
		cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
			AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
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
	r.Mount("/v1/bookings/guest", gh.Routes())
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
