package repository

import (
	"context"
	"time"

	"github.com/diagnosis/luxsuv-bookings/services/bookings/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository interface {
	CreateGuest(ctx context.Context, req *domain.BookingGuestReq) (*domain.Booking, error)
	GetByID(ctx context.Context, id int64) (*domain.Booking, error)
	GetByIDWithToken(ctx context.Context, id int64, token string) (*domain.Booking, error)
	ListByEmail(ctx context.Context, email string, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error)
	ListByUserID(ctx context.Context, userID int64, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error)
	List(ctx context.Context, limit, offset int) ([]domain.Booking, error)
	ListByStatus(ctx context.Context, status domain.BookingStatus, limit, offset int) ([]domain.Booking, error)
	UpdateGuest(ctx context.Context, id int64, token string, patch domain.GuestPatch) (*domain.Booking, error)
	Update(ctx context.Context, id int64, patch domain.GuestPatch) (*domain.Booking, error)
	CancelWithToken(ctx context.Context, id int64, token string) (bool, error)
	Cancel(ctx context.Context, id int64) (bool, error)
	CreateForUser(ctx context.Context, userID int64, req *domain.BookingGuestReq) (*domain.Booking, error)
}

type bookingRepository struct {
	pool *pgxpool.Pool
}

func NewBookingRepository(pool *pgxpool.Pool) BookingRepository {
	return &bookingRepository{pool: pool}
}

const bookingCols = `id, manage_token, status,
rider_name, rider_email, rider_phone,
pickup, dropoff, scheduled_at, notes,
passengers, luggages, ride_type,
user_id, driver_id, reschedule_count, created_at, updated_at`

func (r *bookingRepository) CreateGuest(ctx context.Context, req *domain.BookingGuestReq) (*domain.Booking, error) {
	const q = `INSERT INTO bookings (
		manage_token, status,
		rider_name, rider_email, rider_phone,
		pickup, dropoff, scheduled_at, notes,
		passengers, luggages, ride_type, reschedule_count
	) VALUES ($1,'pending',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,0)
	RETURNING ` + bookingCols

	token := uuid.NewString()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q, token,
		req.RiderName, req.RiderEmail, req.RiderPhone,
		req.Pickup, req.Dropoff, req.ScheduledAt, req.Notes,
		req.Passengers, req.Luggages, req.RideType,
	).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.RescheduleCount,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *bookingRepository) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	const q = `SELECT ` + bookingCols + ` FROM bookings WHERE id=$1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.RescheduleCount,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

func (r *bookingRepository) GetByIDWithToken(ctx context.Context, id int64, token string) (*domain.Booking, error) {
	const q = `SELECT ` + bookingCols + ` FROM bookings WHERE id=$1 AND manage_token=$2`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q, id, token).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.RescheduleCount,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

func (r *bookingRepository) ListByEmail(ctx context.Context, email string, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	q := `SELECT ` + bookingCols + ` FROM bookings WHERE lower(rider_email)=lower($1)`
	args := []any{email}
	if status != nil {
		q += ` AND status=$2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = append(args, *status, limit, offset)
	} else {
		q += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.UserID, &b.DriverID, &b.RescheduleCount,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *bookingRepository) ListByUserID(ctx context.Context, userID int64, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	q := `SELECT ` + bookingCols + ` FROM bookings WHERE user_id=$1`
	args := []any{userID}
	if status != nil {
		q += ` AND status=$2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = append(args, *status, limit, offset)
	} else {
		q += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.UserID, &b.DriverID, &b.RescheduleCount,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *bookingRepository) List(ctx context.Context, limit, offset int) ([]domain.Booking, error) {
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

	var bookings []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.UserID, &b.DriverID, &b.RescheduleCount,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *bookingRepository) ListByStatus(ctx context.Context, status domain.BookingStatus, limit, offset int) ([]domain.Booking, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	const q = `SELECT ` + bookingCols + ` FROM bookings WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.pool.Query(ctx, q, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.UserID, &b.DriverID, &b.RescheduleCount,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (r *bookingRepository) UpdateGuest(ctx context.Context, id int64, token string, patch domain.GuestPatch) (*domain.Booking, error) {
	const q = `
		UPDATE bookings
		SET
			rider_name   = COALESCE($3, rider_name),
			rider_phone  = COALESCE($4, rider_phone),
			pickup       = COALESCE($5, pickup),
			dropoff      = COALESCE($6, dropoff),
			scheduled_at = COALESCE($7, scheduled_at),
			notes        = COALESCE($8, notes),
			passengers   = COALESCE($9, passengers),
			luggages     = COALESCE($10, luggages),
			ride_type    = COALESCE($11, ride_type),
			reschedule_count = CASE 
				WHEN $7 IS NOT NULL AND $7 != scheduled_at THEN reschedule_count + 1
				ELSE reschedule_count
			END,
			updated_at   = now()
		WHERE id=$1 AND manage_token=$2
		RETURNING ` + bookingCols

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q,
		id, token,
		patch.RiderName,
		patch.RiderPhone,
		patch.Pickup,
		patch.Dropoff,
		patch.ScheduledAt,
		patch.Notes,
		patch.Passengers,
		patch.Luggages,
		patch.RideType,
	).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.RescheduleCount,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

func (r *bookingRepository) Update(ctx context.Context, id int64, patch domain.GuestPatch) (*domain.Booking, error) {
	const q = `
		UPDATE bookings
		SET
			rider_name   = COALESCE($2, rider_name),
			rider_phone  = COALESCE($3, rider_phone),
			pickup       = COALESCE($4, pickup),
			dropoff      = COALESCE($5, dropoff),
			scheduled_at = COALESCE($6, scheduled_at),
			notes        = COALESCE($7, notes),
			passengers   = COALESCE($8, passengers),
			luggages     = COALESCE($9, luggages),
			ride_type    = COALESCE($10, ride_type),
			reschedule_count = CASE 
				WHEN $6 IS NOT NULL AND $6 != scheduled_at THEN reschedule_count + 1
				ELSE reschedule_count
			END,
			updated_at   = now()
		WHERE id=$1
		RETURNING ` + bookingCols

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q,
		id,
		patch.RiderName,
		patch.RiderPhone,
		patch.Pickup,
		patch.Dropoff,
		patch.ScheduledAt,
		patch.Notes,
		patch.Passengers,
		patch.Luggages,
		patch.RideType,
	).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.RescheduleCount,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

func (r *bookingRepository) CancelWithToken(ctx context.Context, id int64, token string) (bool, error) {
	const q = `UPDATE bookings SET status='canceled', updated_at=now() WHERE id=$1 AND manage_token=$2 AND status != 'canceled'`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	result, err := r.pool.Exec(ctx, q, id, token)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *bookingRepository) Cancel(ctx context.Context, id int64) (bool, error) {
	const q = `UPDATE bookings SET status='canceled', updated_at=now() WHERE id=$1 AND status != 'canceled'`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	result, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *bookingRepository) CreateForUser(ctx context.Context, userID int64, req *domain.BookingGuestReq) (*domain.Booking, error) {
	const q = `INSERT INTO bookings (
		manage_token, status,
		rider_name, rider_email, rider_phone,
		pickup, dropoff, scheduled_at, notes,
		passengers, luggages, ride_type,
		user_id, reschedule_count
	) VALUES ($1,'pending',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,0)
	RETURNING ` + bookingCols

	token := uuid.NewString()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q, token,
		req.RiderName, req.RiderEmail, req.RiderPhone,
		req.Pickup, req.Dropoff, req.ScheduledAt, req.Notes,
		req.Passengers, req.Luggages, req.RideType,
		userID,
	).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.RescheduleCount,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}