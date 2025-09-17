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
	logger.Info("Starting driver service on :8084")
	
	r := chi.NewRouter()
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("driver"))
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// Placeholder endpoints
	r.Get("/assignments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Driver service - assignments endpoint",
			"status": "not_implemented",
		})
	})
	
	r.Get("/availability", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Driver service - availability endpoint",
			"status": "not_implemented",
		})
	})
	
	if err := http.ListenAndServe(":8084", r); err != nil {
		logger.Error("Driver service error", "error", err)
		os.Exit(1)
	}
}