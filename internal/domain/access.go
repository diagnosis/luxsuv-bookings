package domain

import "time"

type GuestAccessRequest struct {
	Email string `json:"email"`
}

type GuestAccessVerify struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type GuestSessionResponse struct {
	SessionToken string `json:"session_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
type OneTimeBookingSessionResponse struct {
	SessionToken string `json:"session_token"`
	BookingID    int64  `json:"booking_id"`
	ExpiresIn    int64  `json:"expires_in"`
}

type GuestAccessCode struct {
	ID        int64
	Email     string
	CodeHash  string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	Attempts  int
	CreatedAt time.Time
}
