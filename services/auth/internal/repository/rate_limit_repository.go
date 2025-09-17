package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RateLimitRepository interface {
	CheckRateLimit(ctx context.Context, key string, requests int, window time.Duration) (bool, error)
	CleanupExpired(ctx context.Context) (int64, error)
}

type rateLimitRepository struct {
	pool *pgxpool.Pool
}

func NewRateLimitRepository(pool *pgxpool.Pool) RateLimitRepository {
	return &rateLimitRepository{pool: pool}
}

func (r *rateLimitRepository) CheckRateLimit(ctx context.Context, key string, requests int, window time.Duration) (bool, error) {
	// Hash the key for privacy
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hashedKey := fmt.Sprintf("%x", hasher.Sum(nil))
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	now := time.Now()
	windowStart := now.Add(-window)
	
	// Use PostgreSQL UPSERT to atomically check and update rate limit
	query := `
		INSERT INTO rate_limits (rl_key, count, window_start, expires_at)
		VALUES ($1, 1, $2, $3)
		ON CONFLICT (rl_key) DO UPDATE SET
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
	err := r.pool.QueryRow(ctx, query, hashedKey, windowStart, now.Add(time.Hour)).Scan(&count)
	if err != nil {
		// On database error, allow the request (fail open)
		return true, nil
	}
	
	return count <= requests, nil
}

func (r *rateLimitRepository) CleanupExpired(ctx context.Context) (int64, error) {
	const q = `DELETE FROM rate_limits WHERE expires_at < now()`
	
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	result, err := r.pool.Exec(ctx, q)
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected(), nil
}