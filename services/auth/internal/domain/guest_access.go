package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

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

type EmailVerificationToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Validation methods
func (r *GuestAccessRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !isValidEmailFormat(r.Email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func (r *GuestAccessVerify) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !isValidEmailFormat(r.Email) {
		return fmt.Errorf("invalid email format")
	}
	if r.Code == "" {
		return fmt.Errorf("code is required")
	}
	if len(r.Code) != 6 {
		return fmt.Errorf("code must be 6 digits")
	}
	// Check if code contains only digits
	if !regexp.MustCompile(`^\d{6}$`).MatchString(r.Code) {
		return fmt.Errorf("code must contain only digits")
	}
	return nil
}

// Normalize methods
func (r *GuestAccessRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

func (r *GuestAccessVerify) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Code = strings.TrimSpace(r.Code)
}

// Helper function
func isValidEmailFormat(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// Constants
const (
	PurposeEmailLogin        = "email_login"
	PurposeEmailVerification = "email_verification"
	
	MaxVerificationAttempts = 5
	CodeExpirationTime      = 15 * time.Minute
	TokenExpirationTime     = 2 * time.Hour
)

// Business logic methods
func (g *GuestAccessCode) IsExpired() bool {
	return time.Now().After(g.ExpiresAt)
}

func (g *GuestAccessCode) IsUsed() bool {
	return g.UsedAt != nil
}

func (g *GuestAccessCode) CanAttempt() bool {
	return g.Attempts < MaxVerificationAttempts && !g.IsExpired() && !g.IsUsed()
}

func (e *EmailVerificationToken) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

func (e *EmailVerificationToken) IsUsed() bool {
	return e.UsedAt != nil
}

func (e *EmailVerificationToken) IsValid() bool {
	return !e.IsExpired() && !e.IsUsed()
}