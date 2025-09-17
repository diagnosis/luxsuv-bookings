package repository

import (
	"context"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type VerifyRepository interface {
	CreateGuestAccess(ctx context.Context, email, codeHash, magic string, expiresAt time.Time, ip net.IP) error
	CheckGuestCode(ctx context.Context, email, code string) (bool, error)
	ConsumeGuestMagic(ctx context.Context, token string) (string, bool, error)
	DeleteExpiredTokens(ctx context.Context) (int64, error)
}

type verifyRepository struct {
	pool *pgxpool.Pool
}

func NewVerifyRepository(pool *pgxpool.Pool) VerifyRepository {
	return &verifyRepository{pool: pool}
}

func (r *verifyRepository) CreateGuestAccess(ctx context.Context, email, codeHash, magic string, expiresAt time.Time, ip net.IP) error {
	const q = `
		INSERT INTO guest_access_codes(email, code_hash, token, expires_at, ip_created)
		VALUES($1,$2,$3,$4,$5)
	`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, q, email, codeHash, magic, expiresAt, ip)
	return err
}

func (r *verifyRepository) CheckGuestCode(ctx context.Context, email, code string) (bool, error) {
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

func (r *verifyRepository) ConsumeGuestMagic(ctx context.Context, token string) (string, bool, error) {
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

func (r *verifyRepository) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := r.pool.Exec(ctx, `
		DELETE FROM guest_access_codes
		WHERE (used_at IS NOT NULL AND used_at < now() - interval '30 days')
		   OR (used_at IS NULL AND expires_at < now() - interval '30 days')
	`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
