package domain

import "time"

type BookingStatus string

const (
	BookingPending   BookingStatus = "pending"
	BookingConfirmed BookingStatus = "confirmed"
	BookingAssigned  BookingStatus = "assigned"
	BookingOnTrip    BookingStatus = "on_trip"
	BookingCompleted BookingStatus = "completed"
	BookingCanceled  BookingStatus = "canceled"
)

func ParseBookingStatus(s string) (BookingStatus, bool) {
	switch BookingStatus(s) {
	case BookingPending, BookingConfirmed, BookingAssigned, BookingOnTrip, BookingCompleted, BookingCanceled:
		return BookingStatus(s), true
	default:
		return "", false
	}
}

type RideType string

const (
	RidePerRide RideType = "per_ride"
	RideHourly  RideType = "hourly"
)

type Booking struct {
	ID             int64         `json:"id"`
	ManageToken    string        `json:"manage_token"`
	Status         BookingStatus `json:"status"`
	RiderName      string        `json:"rider_name"`
	RiderEmail     string        `json:"rider_email"`
	RiderPhone     string        `json:"rider_phone"`
	Pickup         string        `json:"pickup"`
	Dropoff        string        `json:"dropoff"`
	ScheduledAt    time.Time     `json:"scheduled_at"`
	Notes          string        `json:"notes"`
	Passengers     int           `json:"passengers"`
	Luggages       int           `json:"luggages"`
	RideType       RideType      `json:"ride_type"`
	UserID         *int64        `json:"user_id,omitempty"`
	DriverID       *int64        `json:"driver_id,omitempty"`
	RescheduleCount int          `json:"reschedule_count"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type BookingGuestReq struct {
	RiderName   string    `json:"rider_name"`
	RiderEmail  string    `json:"rider_email"`
	RiderPhone  string    `json:"rider_phone"`
	Pickup      string    `json:"pickup"`
	Dropoff     string    `json:"dropoff"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Notes       string    `json:"notes"`
	Passengers  int       `json:"passengers"`
	Luggages    int       `json:"luggages"`
	RideType    RideType  `json:"ride_type"`
}

type BookingGuestRes struct {
	ID          int64     `json:"id"`
	ManageToken string    `json:"manage_token"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

type BookingDTO struct {
	ID             int64     `json:"id"`
	Status         string    `json:"status"`
	RiderName      string    `json:"rider_name"`
	RiderEmail     string    `json:"rider_email"`
	RiderPhone     string    `json:"rider_phone"`
	Pickup         string    `json:"pickup"`
	Dropoff        string    `json:"dropoff"`
	ScheduledAt    time.Time `json:"scheduled_at"`
	Notes          string    `json:"notes"`
	Passengers     int       `json:"passengers"`
	Luggages       int       `json:"luggages"`
	RideType       string    `json:"ride_type"`
	DriverID       *int64    `json:"driver_id,omitempty"`
	RescheduleCount int      `json:"reschedule_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	UserID         *int64    `json:"user_id,omitempty"`
}

type GuestPatch struct {
	RiderName   *string    `json:"rider_name,omitempty"`
	RiderPhone  *string    `json:"rider_phone,omitempty"`
	Pickup      *string    `json:"pickup,omitempty"`
	Dropoff     *string    `json:"dropoff,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	Passengers  *int       `json:"passengers,omitempty"`
	Luggages    *int       `json:"luggages,omitempty"`
	RideType    *RideType  `json:"ride_type,omitempty"`
}

// Business Rules
const (
	MaxRescheduleCount = 2
	CancelCutoffHours  = 24
	MaxPassengers      = 8
	MinPassengers      = 1
	MaxLuggages        = 10
	MinLuggages        = 0
)

// CanReschedule checks if booking can be rescheduled
func (b *Booking) CanReschedule() bool {
	return b.RescheduleCount < MaxRescheduleCount && 
		   b.Status != BookingCanceled && 
		   b.Status != BookingCompleted
}

// CanCancel checks if booking can be canceled based on 24h rule
func (b *Booking) CanCancel() bool {
	if b.Status == BookingCanceled || b.Status == BookingCompleted {
		return false
	}
	
	cutoffTime := b.ScheduledAt.Add(-CancelCutoffHours * time.Hour)
	return time.Now().Before(cutoffTime)
}

// IsOwner checks if the given email owns this booking
func (b *Booking) IsOwner(email string) bool {
	return strings.EqualFold(b.RiderEmail, email)
}

// IsUserOwner checks if the given user ID owns this booking
func (b *Booking) IsUserOwner(userID int64) bool {
	return b.UserID != nil && *b.UserID == userID
}