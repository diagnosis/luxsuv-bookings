package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID           int64
	Role         string
	Email        string
	PasswordHash string
	Name         string
	Phone        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UsersRepo interface {
	Create(ctx context.Context, email, hash, name, phone string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	LinkExistingBookings(ctx context.Context, userID int64, email string) error
}

type UsersRepoImpl struct{ pool *pgxpool.Pool }

func NewUsersRepo(pool *pgxpool.Pool) *UsersRepoImpl { return &UsersRepoImpl{pool: pool} }

func (r *UsersRepoImpl) Create(ctx context.Context, email, hash, name, phone string) (*User, error) {
	const q = `
INSERT INTO users (email, password_hash, name, phone)
VALUES ($1,$2,$3,$4)
RETURNING id, role, email, password_hash, name, phone, created_at, updated_at`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var u User
	if err := r.pool.QueryRow(ctx, q, email, hash, name, phone).Scan(
		&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsersRepoImpl) FindByEmail(ctx context.Context, email string) (*User, error) {
	const q = `SELECT id, role, email, password_hash, name, phone, created_at, updated_at FROM users WHERE email=$1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var u User
	if err := r.pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsersRepoImpl) FindByID(ctx context.Context, id int64) (*User, error) {
	const q = `SELECT id, role, email, password_hash, name, phone, created_at, updated_at FROM users WHERE id=$1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var u User
	if err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UsersRepoImpl) LinkExistingBookings(ctx context.Context, userID int64, email string) error {
	const q = `UPDATE bookings SET user_id=$1 WHERE rider_email=$2 AND user_id IS NULL`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, q, userID, email)
	return err
}
