package repository

import (
	"context"
	"net"
	"time"

	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type VerifyRepository interface {
	// Email verification tokens
	CreateEmailVerification(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	ConsumeEmailVerification(ctx context.Context, token string) (userID int64, err error)
	DeleteExpiredTokens(ctx context.Context) (int64, error)
	
	// Guest access codes
	CreateGuestAccess(ctx context.Context, email, codeHash, magic string, expiresAt time.Time, ip net.IP) error
	CheckGuestCode(ctx context.Context, email, code string) (bool, error)
	ConsumeGuestMagic(ctx context.Context, token string) (email string, valid bool, err error)
	IncrementGuestAttempts(ctx context.Context, email string) error
}

type verifyRepository struct {
	pool *pgxpool.Pool
}

func NewVerifyRepository(pool *pgxpool.Pool) VerifyRepository {
	return &verifyRepository{pool: pool}
}

// Email verification tokens
func (r *verifyRepository) CreateEmailVerification(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	const q = `
		INSERT INTO email_verification_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)`
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(ctx, q, userID, token, expiresAt)
	return err
}

func (r *verifyRepository) ConsumeEmailVerification(ctx context.Context, token string) (int64, error) {
	const q = `
		UPDATE email_verification_tokens
		SET used_at = now()
		WHERE token = $1
		  AND used_at IS NULL
		  AND expires_at > now()
		RETURNING user_id`
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	var userID int64
	err := r.pool.QueryRow(ctx, q, token).Scan(&userID)
	if err == pgx.ErrNoRows {
		return 0, nil // Invalid, used, or expired
	}
	return userID, err
}

func (r *verifyRepository) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	const q = `
		DELETE FROM email_verification_tokens
		WHERE (used_at IS NOT NULL AND used_at < now() - interval '30 days')
		   OR (used_at IS NULL AND expires_at < now() - interval '7 days')`
	
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	result, err := r.pool.Exec(ctx, q)
	if err != nil {
		return 0, err
	}
	
	return result.RowsAffected(), nil
}

// Guest access codes
func (r *verifyRepository) CreateGuestAccess(ctx context.Context, email, codeHash, magic string, expiresAt time.Time, ip net.IP) error {
	const q = `
		INSERT INTO guest_access_codes (email, code_hash, token, expires_at, ip_created)
		VALUES ($1, $2, $3, $4, $5)`
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(ctx, q, email, codeHash, magic, expiresAt, ip)
	return err
}

func (r *verifyRepository) CheckGuestCode(ctx context.Context, email, code string) (bool, error) {
	const q = `
		SELECT id, code_hash, expires_at, used_at, attempts
		FROM guest_access_codes
		WHERE lower(email) = lower($1)
		ORDER BY id DESC
		LIMIT 1`
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	var (
		id       int64
		hash     string
		expires  time.Time
		used     *time.Time
		attempts int
	)
	
	err := r.pool.QueryRow(ctx, q, email).Scan(&id, &hash, &expires, &used, &attempts)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	
	// Check if expired, used, or too many attempts
	if used != nil || time.Now().After(expires) || attempts >= domain.MaxVerificationAttempts {
		return false, nil
	}
	
	// Verify the code
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)); err != nil {
		// Increment attempts on wrong code
		_, _ = r.pool.Exec(ctx, `UPDATE guest_access_codes SET attempts = attempts + 1 WHERE id = $1`, id)
		return false, nil
	}
	
	// Mark as used
	_, _ = r.pool.Exec(ctx, `UPDATE guest_access_codes SET used_at = now() WHERE id = $1`, id)
	return true, nil
}

func (r *verifyRepository) ConsumeGuestMagic(ctx context.Context, token string) (string, bool, error) {
	const q = `
		SELECT id, email, expires_at, used_at
		FROM guest_access_codes
		WHERE token = $1`
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	var (
		id      int64
		email   string
		expires time.Time
		used    *time.Time
	)
	
	err := r.pool.QueryRow(ctx, q, token).Scan(&id, &email, &expires, &used)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	
	// Check if expired or used
	if used != nil || time.Now().After(expires) {
		return "", false, nil
	}
	
	// Mark as used
	_, _ = r.pool.Exec(ctx, `UPDATE guest_access_codes SET used_at = now() WHERE id = $1`, id)
	return email, true, nil
}

func (r *verifyRepository) IncrementGuestAttempts(ctx context.Context, email string) error {
	const q = `
		UPDATE guest_access_codes
		SET attempts = attempts + 1
		WHERE lower(email) = lower($1)
		  AND used_at IS NULL
		  AND expires_at > now()`
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(ctx, q, email)
	return err
}