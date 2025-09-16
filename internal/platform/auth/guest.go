package auth

import "time"

func NewGuestSession(email string, ttl time.Duration) (string, error) {
	return NewAccessToken(0, email, "guest", "guest.bookings:read guest.bookings:write", ttl)
}
