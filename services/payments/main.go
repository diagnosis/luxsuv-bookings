package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
	mw "github.com/diagnosis/luxsuv-bookings/pkg/middleware"
)

func main() {
	logger.Info("Starting payments service on :8085")
	
	r := chi.NewRouter()
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("payments"))
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// Placeholder endpoints
	r.Post("/intent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Payments service - create intent endpoint",
			"status": "not_implemented",
		})
	})
	
	r.Post("/webhook", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Payments service - webhook endpoint",
			"status": "not_implemented",
		})
	})
	
	if err := http.ListenAndServe(":8085", r); err != nil {
		logger.Error("Payments service error", "error", err)
		os.Exit(1)
	}
}