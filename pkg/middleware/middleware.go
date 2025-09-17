package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/diagnosis/luxsuv-bookings/pkg/logger"
)

// RequestID adds a unique request ID to each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		ctx := context.WithValue(r.Context(), logger.RequestIDKey, requestID)
		w.Header().Set("X-Request-ID", requestID)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging logs HTTP requests with structured logging
func Logging(next http.Handler) http.Handler {
	return middleware.RequestLogger(&StructuredLogger{})(next)
}

type StructuredLogger struct{}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &StructuredLogEntry{
		request: r,
		start:   time.Now(),
	}
}

type StructuredLogEntry struct {
	request *http.Request
	start   time.Time
}

func (l *StructuredLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	logger.InfoContext(l.request.Context(), "HTTP request completed",
		"method", l.request.Method,
		"path", l.request.URL.Path,
		"status", status,
		"bytes", bytes,
		"elapsed_ms", elapsed.Milliseconds(),
		"user_agent", l.request.UserAgent(),
		"remote_addr", l.request.RemoteAddr,
	)
}

func (l *StructuredLogEntry) Panic(v interface{}, stack []byte) {
	logger.ErrorContext(l.request.Context(), "HTTP request panic",
		"panic", v,
		"stack", string(stack),
		"method", l.request.Method,
		"path", l.request.URL.Path,
	)
}

// CORS handles Cross-Origin Resource Sharing
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, Idempotency-Key")
		w.Header().Set("Access-Control-Expose-Headers", "Link")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "300")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ServiceName adds service name to context for logging
func ServiceName(name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), logger.ServiceKey, name)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Metrics provides Prometheus metrics endpoint
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			// TODO: Implement Prometheus metrics
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Metrics endpoint - TODO: implement Prometheus metrics\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Health provides health check endpoint
func Health(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// IdempotencyKey handles idempotency for POST requests
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

func IdempotencyMiddleware(store IdempotencyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Hash the key for privacy
			hasher := sha256.New()
			hasher.Write([]byte(key))
			hashedKey := fmt.Sprintf("idempotency:%x", hasher.Sum(nil))

			// Check if we've seen this key before
			if existing, err := store.Get(r.Context(), hashedKey); err == nil && existing != "" {
				// Return cached response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(existing))
				return
			}

			// Capture response for caching
			recorder := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)

			// Cache successful responses
			if recorder.statusCode >= 200 && recorder.statusCode < 300 {
				store.Set(r.Context(), hashedKey, string(recorder.body), 24*time.Hour)
			}
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	r.body = append(r.body, body...)
	return r.ResponseWriter.Write(body)
}