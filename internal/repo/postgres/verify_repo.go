package postgres

import (
	"context"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// VerifyRepo defines operations for email verification tokens and user verification flags.
type VerifyRepo interface {
	// CreateEmailVerification inserts a one-time verification token for a user.
	CreateEmailVerification(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	// ConsumeEmailVerification marks a token used if valid, and returns the userID (0 if not found/invalid/expired/used).
	ConsumeEmailVerification(ctx context.Context, token string) (userID int64, err error)
	// MarkUserVerified sets users.is_verified = true.
	MarkUserVerified(ctx context.Context, userID int64) error
	// IsUserVerified returns whether a user is verified.
	IsUserVerified(ctx context.Context, userID int64) (bool, error)

	// Optional helpers (nice to have)
	// DeleteExpiredTokens removes old/expired tokens (maintenance).
	DeleteExpiredTokens(ctx context.Context) (int64, error)

	// guest access:
	CreateGuestAccess(ctx context.Context, email, codeHash, magic string, expiresAt time.Time, ip net.IP) error
	CheckGuestCode(ctx context.Context, email, code string) (bool, error)
	ConsumeGuestMagic(ctx context.Context, token string) (string, bool, error)
}

type VerifyRepoImpl struct{ pool *pgxpool.Pool }

func NewVerifyRepo(pool *pgxpool.Pool) *VerifyRepoImpl { return &VerifyRepoImpl{pool: pool} }

func (r *VerifyRepoImpl) CreateEmailVerification(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// You can keep multiple tokens per user; the most recent is usually sent in email.
	// If you prefer only one active at a time, add: DELETE FROM ... WHERE user_id=$1 AND used_at IS NULL
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verification_tokens (user_id, token, expires_at)
         VALUES ($1, $2, $3)`,
		userID, token, expiresAt,
	)
	return err
}

func (r *VerifyRepoImpl) ConsumeEmailVerification(ctx context.Context, token string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var userID int64
	// Mark-used *and* return user id atomically, only if not used and not expired.
	err := r.pool.QueryRow(ctx, `
UPDATE email_verification_tokens
SET used_at = now()
WHERE token = $1
  AND used_at IS NULL
  AND expires_at > now()
RETURNING user_id
`, token).Scan(&userID)

	if err == pgx.ErrNoRows {
		return 0, nil // invalid, used or expired
	}
	return userID, err
}

func (r *VerifyRepoImpl) MarkUserVerified(ctx context.Context, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := r.pool.Exec(ctx, `
UPDATE users
SET is_verified = true, updated_at = now()
WHERE id = $1
`, userID)
	return err
}

func (r *VerifyRepoImpl) IsUserVerified(ctx context.Context, userID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var verified bool
	err := r.pool.QueryRow(ctx, `
SELECT is_verified FROM users WHERE id = $1
`, userID).Scan(&verified)

	if err == pgx.ErrNoRows {
		// unknown user; you can choose to return false,nil or an error.
		return false, nil
	}
	return verified, err
}

func (r *VerifyRepoImpl) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := r.pool.Exec(ctx, `
DELETE FROM email_verification_tokens
WHERE (used_at IS NOT NULL AND used_at < now() - interval '30 days')
   OR (used_at IS NULL AND expires_at < now() - interval '30 days')
`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
func (r *VerifyRepoImpl) CreateGuestAccess(ctx context.Context, email, codeHash, magic string, expiresAt time.Time, ip net.IP) error {
	const q = `
		INSERT INTO guest_access_codes(email, code_hash, token, expires_at, ip_created)
		VALUES($1,$2,$3,$4,$5)
	`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, q, email, codeHash, magic, expiresAt, ip)
	return err
}

func (r *VerifyRepoImpl) CheckGuestCode(ctx context.Context, email, code string) (bool, error) {
	const q = `
		SELECT id, code_hash, expires_at, used_at
		FROM guest_access_codes
		WHERE lower(email)=lower($1)
		ORDER BY id DESC
		LIMIT 1
	`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var (
		id      int64
		hash    string
		expires time.Time
		used    *time.Time
	)
	err := r.pool.QueryRow(ctx, q, email).Scan(&id, &hash, &expires, &used)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if used != nil || time.Now().After(expires) {
		return false, nil
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)); err != nil {
		// bump attempts
		_, _ = r.pool.Exec(ctx, `UPDATE guest_access_codes SET attempts=attempts+1 WHERE id=$1`, id)
		return false, nil
	}
	_, _ = r.pool.Exec(ctx, `UPDATE guest_access_codes SET used_at=now() WHERE id=$1`, id)
	return true, nil
}

func (r *VerifyRepoImpl) ConsumeGuestMagic(ctx context.Context, token string) (string, bool, error) {
	const q = `
		SELECT id, email, expires_at, used_at
		FROM guest_access_codes
		WHERE token=$1
	`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var (
		id      int64
		email   string
		expires time.Time
		used    *time.Time
	)
	if err := r.pool.QueryRow(ctx, q, token).Scan(&id, &email, &expires, &used); err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	if used != nil || time.Now().After(expires) {
		return "", false, nil
	}
	_, _ = r.pool.Exec(ctx, `UPDATE guest_access_codes SET used_at=now() WHERE id=$1`, id)
	return email, true, nil
}
