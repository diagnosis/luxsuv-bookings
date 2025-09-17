package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/http/response"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	Requests      int           // Max requests per window
	Window        time.Duration // Time window duration
	KeyFunc       func(r *http.Request) []string // Function to generate rate limit keys
	SkipFunc      func(r *http.Request) bool     // Function to skip rate limiting
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	pool   *pgxpool.Pool
	config RateLimitConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(pool *pgxpool.Pool, config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		pool:   pool,
		config: config,
	}
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting if skip function returns true
			if rl.config.SkipFunc != nil && rl.config.SkipFunc(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Get rate limit keys (IP, email, etc.)
			keys := rl.config.KeyFunc(r)
			
			// Check rate limits for all keys
			for _, key := range keys {
				if !rl.checkRateLimit(r.Context(), key) {
					response.RateLimit(w, fmt.Sprintf("Too many requests. Try again later."))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit checks if a request is within rate limits
func (rl *RateLimiter) checkRateLimit(ctx context.Context, key string) bool {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Hash the key for privacy
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hashedKey := fmt.Sprintf("%x", hasher.Sum(nil))

	now := time.Now()
	windowStart := now.Add(-rl.config.Window)

	// Use PostgreSQL UPSERT to atomically check and update rate limit
	query := `
		INSERT INTO rate_limits (key, count, window_start, expires_at)
		VALUES ($1, 1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET
			count = CASE 
				WHEN rate_limits.window_start < $2 THEN 1
				ELSE rate_limits.count + 1
			END,
			window_start = CASE
				WHEN rate_limits.window_start < $2 THEN $2
				ELSE rate_limits.window_start
			END,
			expires_at = $3
		RETURNING count`

	var count int
	err := rl.pool.QueryRow(ctx, query, hashedKey, windowStart, now.Add(time.Hour)).Scan(&count)
	if err != nil {
		// On database error, allow the request (fail open)
		return true
	}

	return count <= rl.config.Requests
}

// GuestAccessRateLimitKeyFunc generates rate limit keys for guest access requests
func GuestAccessRateLimitKeyFunc(r *http.Request) []string {
	keys := []string{}
	
	// Rate limit by IP
	ip := getClientIP(r)
	if ip != "" {
		keys = append(keys, "ip:"+ip)
	}
	
	// Rate limit by email if provided in request body
	if r.Method == "POST" {
		// We'll read the email from the request body in the handler
		// For now, just use IP-based rate limiting
	}
	
	return keys
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if there are multiple
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	
	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}