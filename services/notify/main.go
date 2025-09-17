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
	logger.Info("Starting notify service on :8086")
	
	r := chi.NewRouter()
	r.Use(mw.RequestID)
	r.Use(mw.ServiceName("notify"))
	r.Use(mw.Logging)
	r.Use(mw.CORS)
	r.Use(mw.Health)
	r.Use(mw.Metrics)
	
	// Placeholder endpoints
	r.Post("/send", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Notify service - send notification endpoint",
			"status": "not_implemented",
		})
	})
	
	if err := http.ListenAndServe(":8086", r); err != nil {
		logger.Error("Notify service error", "error", err)
		os.Exit(1)
	}
}