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
	logger.Info("Starting dispatch service on :8083")
	
	r := chi.NewRouter()
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("dispatch"))
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// Placeholder endpoints
	r.Get("/pending", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Dispatch service - pending bookings endpoint",
			"status": "not_implemented",
		})
	})
	
	r.Post("/assign", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Dispatch service - assign booking endpoint",
			"status": "not_implemented",
		})
	})
	
	if err := http.ListenAndServe(":8083", r); err != nil {
		logger.Error("Dispatch service error", "error", err)
		os.Exit(1)
	}
}