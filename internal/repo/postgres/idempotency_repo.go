package postgres

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyRepo handles idempotency operations
type IdempotencyRepo interface {
	// CheckOrCreateIdempotency checks if an idempotency key exists and returns the booking ID
	// If it doesn't exist, returns 0. If it exists, returns the existing booking ID.
	CheckOrCreateIdempotency(ctx context.Context, key string, bookingID int64) (existingBookingID int64, err error)
	// CleanupExpired removes expired idempotency records
	CleanupExpired(ctx context.Context) (int64, error)
}

type IdempotencyRepoImpl struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepo(pool *pgxpool.Pool) *IdempotencyRepoImpl {
	return &IdempotencyRepoImpl{pool: pool}
}

func (r *IdempotencyRepoImpl) CheckOrCreateIdempotency(ctx context.Context, key string, bookingID int64) (int64, error) {
	// Hash the idempotency key for privacy and consistent length
	hasher := sha256.New()
	hasher.Write([]byte(key))
	keyHash := fmt.Sprintf("%x", hasher.Sum(nil))

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// First, try to find existing idempotency record
	var existingBookingID int64
	checkQuery := `SELECT booking_id FROM booking_idempotency WHERE key_hash = $1`
	err := r.pool.QueryRow(ctx, checkQuery, keyHash).Scan(&existingBookingID)
	
	if err == nil {
		// Found existing record, return the existing booking ID
		return existingBookingID, nil
	}
	
	if err != pgx.ErrNoRows {
		// Database error
		return 0, err
	}

	// No existing record, create new one if bookingID is provided
	if bookingID > 0 {
		insertQuery := `
			INSERT INTO booking_idempotency (key_hash, booking_id, expires_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (key_hash) DO NOTHING`
		
		expiresAt := time.Now().Add(24 * time.Hour)
		_, err = r.pool.Exec(ctx, insertQuery, keyHash, bookingID, expiresAt)
		if err != nil {
			return 0, err
		}
	}

	return 0, nil
}

func (r *IdempotencyRepoImpl) CleanupExpired(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `DELETE FROM booking_idempotency WHERE expires_at < now()`
	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

var _ IdempotencyRepo = (*IdempotencyRepoImpl)(nil)