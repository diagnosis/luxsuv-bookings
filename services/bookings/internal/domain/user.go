package domain

import "time"

type User struct {
	ID           int64     `json:"id"`
	Role         string    `json:"role"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	IsVerified   bool      `json:"is_verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type GuestAccessCode struct {
	ID        int64      `json:"id"`
	Email     string     `json:"email"`
	CodeHash  string     `json:"-"`
	Token     string     `json:"token"`
	Purpose   string     `json:"purpose"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	Attempts  int        `json:"attempts"`
	CreatedAt time.Time  `json:"created_at"`
}

// Import the missing strings package
import "strings"