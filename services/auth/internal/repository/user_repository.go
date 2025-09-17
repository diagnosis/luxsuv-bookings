package repository

import (
	"context"
	"time"

	"github.com/diagnosis/luxsuv-bookings/services/auth/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, req *domain.CreateUserRequest, passwordHash string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
	Update(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]domain.User, error)
	LinkExistingBookings(ctx context.Context, userID int64, email string) error
	MarkVerified(ctx context.Context, userID int64) error
	UpdateRole(ctx context.Context, userID int64, role string) error
}

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

const userCols = `id, role, email, password_hash, name, phone, is_verified, created_at, updated_at`

func (r *userRepository) Create(ctx context.Context, req *domain.CreateUserRequest, passwordHash string) (*domain.User, error) {
	const q = `
		INSERT INTO users (role, email, password_hash, name, phone, is_verified)
		VALUES ($1, $2, $3, $4, $5, false)
		RETURNING ` + userCols
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	var u domain.User
	err := r.pool.QueryRow(ctx, q, req.Role, req.Email, passwordHash, req.Name, req.Phone).Scan(
		&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	return &u, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `SELECT ` + userCols + ` FROM users WHERE email = $1`
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
	const q = `SELECT ` + userCols + ` FROM users WHERE id = $1`
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

func (r *userRepository) Update(ctx context.Context, id int64, req *domain.UpdateUserRequest) (*domain.User, error) {
	const q = `
		UPDATE users 
		SET 
			name = COALESCE($2, name),
			phone = COALESCE($3, phone),
			role = COALESCE($4, role),
			updated_at = now()
		WHERE id = $1
		RETURNING ` + userCols
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	var u domain.User
	err := r.pool.QueryRow(ctx, q, id, req.Name, req.Phone, req.Role).Scan(
		&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &u, err
}

func (r *userRepository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM users WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	result, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	
	return nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]domain.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	
	const q = `
		SELECT ` + userCols + `
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Role, &u.Email, &u.PasswordHash, &u.Name, &u.Phone, &u.IsVerified, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	
	return users, rows.Err()
}

func (r *userRepository) LinkExistingBookings(ctx context.Context, userID int64, email string) error {
	const q = `UPDATE bookings SET user_id = $1 WHERE rider_email = $2 AND user_id IS NULL`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	_, err := r.pool.Exec(ctx, q, userID, email)
	return err
}

func (r *userRepository) MarkVerified(ctx context.Context, userID int64) error {
	const q = `UPDATE users SET is_verified = true, updated_at = now() WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	result, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return err
	}
	
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	
	return nil
}

func (r *userRepository) UpdateRole(ctx context.Context, userID int64, role string) error {
	const q = `UPDATE users SET role = $2, updated_at = now() WHERE id = $1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	
	result, err := r.pool.Exec(ctx, q, userID, role)
	if err != nil {
		return err
	}
	
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	
	return nil
}