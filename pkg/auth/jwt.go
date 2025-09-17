package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Sub   int64  `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Scope string `json:"scope"`
	jwt.RegisteredClaims
}

func NewAccessToken(sub int64, email, role, scope, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Sub:   sub,
		Email: email,
		Role:  role,
		Scope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Audience:  []string{"luxsuv-api"},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func Parse(tokenString, secret string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tok.Claims.(*Claims); ok && tok.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func NewGuestSession(email, secret string, ttl time.Duration) (string, error) {
	return NewAccessToken(0, email, "guest", "guest.bookings:read guest.bookings:write", secret, ttl)
}