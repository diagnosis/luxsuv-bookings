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
	ID          int64         `json:"id"`
	ManageToken string        `json:"manage_token"`
	Status      BookingStatus `json:"status"`

	RiderName  string `json:"rider_name"`
	RiderEmail string `json:"rider_email"`
	RiderPhone string `json:"rider_phone"`

	Pickup      string    `json:"pickup"`
	Dropoff     string    `json:"dropoff"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Notes       string    `json:"notes"`

	Passengers int      `json:"passengers"`
	Luggages   int      `json:"luggages"`
	RideType   RideType `json:"ride_type"`

	DriverID  *int64    `json:"driver_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID          int64     `json:"id"`
	Status      string    `json:"status"`
	RiderName   string    `json:"rider_name"`
	RiderEmail  string    `json:"rider_email"`
	RiderPhone  string    `json:"rider_phone"`
	Pickup      string    `json:"pickup"`
	Dropoff     string    `json:"dropoff"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Notes       string    `json:"notes"`
	Passengers  int       `json:"passengers"`
	Luggages    int       `json:"luggages"`
	RideType    string    `json:"ride_type"`
	DriverID    *int64    `json:"driver_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
