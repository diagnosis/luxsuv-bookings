package postgres

import (
	"context"
	"time"

	"github.com/diagnosis/luxsuv-bookings/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepo interface {
	CreateGuest(ctx context.Context, in *domain.BookingGuestReq) (*domain.Booking, error)
	GetByIDWithToken(ctx context.Context, id int64, token string) (*domain.Booking, error)
	CancelWithToken(ctx context.Context, id int64, token string) (bool, error)
	List(ctx context.Context, limit, offset int) ([]domain.Booking, error)
	ListByStatus(ctx context.Context, status domain.BookingStatus, limit, offset int) ([]domain.Booking, error)
}

type BookingRepoImpl struct{ pool *pgxpool.Pool }

func NewBookingRepo(pool *pgxpool.Pool) *BookingRepoImpl { return &BookingRepoImpl{pool: pool} }

const bookingCols = `id, manage_token, status,
rider_name, rider_email, rider_phone,
pickup, dropoff, scheduled_at, notes,
passengers, luggages, ride_type,
driver_id, created_at, updated_at`

func (r *BookingRepoImpl) CreateGuest(ctx context.Context, in *domain.BookingGuestReq) (*domain.Booking, error) {
	const q = `INSERT INTO bookings (
    manage_token, status,
    rider_name, rider_email, rider_phone,
    pickup, dropoff, scheduled_at, notes,
    passengers, luggages, ride_type
  ) VALUES ($1,'pending',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
  RETURNING ` + bookingCols

	tok := uuid.NewString()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q, tok,
		in.RiderName, in.RiderEmail, in.RiderPhone,
		in.Pickup, in.Dropoff, in.ScheduledAt, in.Notes,
		in.Passengers, in.Luggages, in.RideType,
	).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.DriverID, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BookingRepoImpl) GetByIDWithToken(ctx context.Context, id int64, token string) (*domain.Booking, error) {
	const q = `SELECT ` + bookingCols + ` FROM bookings WHERE id=$1 AND manage_token=$2`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q, id, token).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.DriverID, &b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

func (r *BookingRepoImpl) CancelWithToken(ctx context.Context, id int64, token string) (bool, error) {
	const q = `UPDATE bookings SET status='canceled' WHERE id=$1 AND manage_token=$2 AND status <> 'canceled'`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ct, err := r.pool.Exec(ctx, q, id, token)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

func (r *BookingRepoImpl) List(ctx context.Context, limit, offset int) ([]domain.Booking, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	const q = `SELECT ` + bookingCols + ` FROM bookings ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bs := make([]domain.Booking, 0, limit)
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.DriverID, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	return bs, rows.Err()
}
func (r *BookingRepoImpl) ListByStatus(ctx context.Context, status domain.BookingStatus, limit, offset int) ([]domain.Booking, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	const q = `
		SELECT ` + bookingCols + `
		FROM bookings
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(ctx, q, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bs := make([]domain.Booking, 0, limit)
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.DriverID, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	return bs, rows.Err()
}

var _ BookingRepo = (*BookingRepoImpl)(nil)
