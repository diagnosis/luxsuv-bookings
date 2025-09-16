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
	GetByID(ctx context.Context, id int64) (*domain.Booking, error)
	ListByUserID(ctx context.Context, userID int64, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error)
	CreateForUser(ctx context.Context, userID int64, in *domain.BookingGuestReq) (*domain.Booking, error)
	UpdateGuest(ctx context.Context, id int64, token string, patch domain.GuestPatch) (*domain.Booking, error)
	ListByEmail(ctx context.Context, email string, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error)
}

type BookingRepoImpl struct{ pool *pgxpool.Pool }

func NewBookingRepo(pool *pgxpool.Pool) *BookingRepoImpl { return &BookingRepoImpl{pool: pool} }

const bookingCols = `id, manage_token, status,
rider_name, rider_email, rider_phone,
pickup, dropoff, scheduled_at, notes,
passengers, luggages, ride_type,
user_id, driver_id, created_at, updated_at`

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
		&b.UserID, // ← add (nullable: *int64 on the struct)
		&b.DriverID,
		&b.CreatedAt, &b.UpdatedAt,
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
		&b.UserID, // ← add (nullable: *int64 on the struct)
		&b.DriverID,
		&b.CreatedAt, &b.UpdatedAt,
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
			&b.UserID, // ← add (nullable: *int64 on the struct)
			&b.DriverID,
			&b.CreatedAt, &b.UpdatedAt,
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
			&b.UserID, // ← add (nullable: *int64 on the struct)
			&b.DriverID,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bs = append(bs, b)
	}
	return bs, rows.Err()
}
func (r *BookingRepoImpl) GetByID(ctx context.Context, id int64) (*domain.Booking, error) {
	const q = `SELECT ` + bookingCols + ` FROM bookings WHERE id=$1`
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var b domain.Booking
	if err := r.pool.QueryRow(ctx, q, id).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, // ← add (nullable: *int64 on the struct)
		&b.DriverID,
		&b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}
func (r *BookingRepoImpl) ListByUserID(ctx context.Context, userID int64, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	base := `SELECT ` + bookingCols + ` FROM bookings WHERE user_id=$1`
	args := []any{userID}
	if status != nil {
		base += ` AND status=$2`
		args = append(args, *status)
		base += ` ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = append(args, limit, offset)
	} else {
		base += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Booking, 0, limit)
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.UserID, // ← add (nullable: *int64 on the struct)
			&b.DriverID,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (r *BookingRepoImpl) CreateForUser(ctx context.Context, userID int64, in *domain.BookingGuestReq) (*domain.Booking, error) {
	const q = `INSERT INTO bookings (
        manage_token, status,
        rider_name, rider_email, rider_phone,
        pickup, dropoff, scheduled_at, notes,
        passengers, luggages, ride_type,
        user_id
    ) VALUES ($1,'pending',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
    RETURNING ` + bookingCols

	tok := uuid.NewString()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q, tok,
		in.RiderName, in.RiderEmail, in.RiderPhone,
		in.Pickup, in.Dropoff, in.ScheduledAt, in.Notes,
		in.Passengers, in.Luggages, in.RideType,
		userID,
	).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
func (r *BookingRepoImpl) ListByEmail(ctx context.Context, email string, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error) {
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

	var out []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(
			&b.ID, &b.ManageToken, &b.Status,
			&b.RiderName, &b.RiderEmail, &b.RiderPhone,
			&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
			&b.Passengers, &b.Luggages, &b.RideType,
			&b.UserID, // ← add (nullable: *int64 on the struct)
			&b.DriverID,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *BookingRepoImpl) UpdateGuest(ctx context.Context, id int64, token string, p domain.GuestPatch) (*domain.Booking, error) {
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
            updated_at   = now()
        WHERE id=$1 AND manage_token=$2
        RETURNING ` + bookingCols

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var b domain.Booking
	err := r.pool.QueryRow(ctx, q,
		id, token,
		p.RiderName,   // $3  *string
		p.RiderPhone,  // $4  *string
		p.Pickup,      // $5  *string
		p.Dropoff,     // $6  *string
		p.ScheduledAt, // $7  *time.Time
		p.Notes,       // $8  *string
		p.Passengers,  // $9  *int
		p.Luggages,    // $10 *int
		p.RideType,    // $11 *domain.RideType
	).Scan(
		&b.ID, &b.ManageToken, &b.Status,
		&b.RiderName, &b.RiderEmail, &b.RiderPhone,
		&b.Pickup, &b.Dropoff, &b.ScheduledAt, &b.Notes,
		&b.Passengers, &b.Luggages, &b.RideType,
		&b.UserID, &b.DriverID, &b.CreatedAt, &b.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &b, err
}

var _ BookingRepo = (*BookingRepoImpl)(nil)
