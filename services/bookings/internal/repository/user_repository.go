package repository

import (
	"context"
	"time"

	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	LinkExistingBookings(ctx context.Context, userID int64, email string) error
}

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `SELECT id, role, email, password_hash, name, phone, is_verified, created_at, updated_at FROM users WHERE email=$1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	var u domain.User
	err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *userRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	const q = `SELECT id, role, email, password_hash, name, phone, is_verified, created_at, updated_at FROM users WHERE id=$1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	var u domain.User
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *userRepository) LinkExistingBookings(ctx context.Context, userID int64, email string) error {
	const q = `UPDATE bookings SET user_id=$1 WHERE rider_email=$2 AND user_id IS NULL`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, q, userID, email)
	return err
}