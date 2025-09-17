package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

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

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Role     string `json:"role,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

type UserInfo struct {
	ID         int64  `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	IsVerified bool   `json:"is_verified"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type UpdateUserRequest struct {
	Name  *string `json:"name,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Role  *string `json:"role,omitempty"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// Valid user roles
const (
	RoleGuest      = "guest"
	RoleRider      = "rider"
	RoleDriver     = "driver"
	RoleDispatcher = "dispatcher"
	RoleAdmin      = "admin"
)

var validRoles = map[string]bool{
	RoleGuest:      true,
	RoleRider:      true,
	RoleDriver:     true,
	RoleDispatcher: true,
	RoleAdmin:      true,
}

// Validation methods
func (r *CreateUserRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !isValidEmail(r.Email) {
		return fmt.Errorf("invalid email format")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(r.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if !isValidPhone(r.Phone) {
		return fmt.Errorf("invalid phone format")
	}
	if r.Role != "" && !validRoles[r.Role] {
		return fmt.Errorf("invalid role")
	}
	return nil
}

func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !isValidEmail(r.Email) {
		return fmt.Errorf("invalid email format")
	}
	if r.Password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

func (r *UpdateUserRequest) Validate() error {
	if r.Role != nil && !validRoles[*r.Role] {
		return fmt.Errorf("invalid role")
	}
	if r.Phone != nil && !isValidPhone(*r.Phone) {
		return fmt.Errorf("invalid phone format")
	}
	return nil
}

func IsValidRole(role string) bool {
	return validRoles[role]
}

// Helper functions
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isValidPhone(phone string) bool {
	// Basic phone validation - starts with + or digit, contains only digits, spaces, hyphens, parentheses
	phoneRegex := regexp.MustCompile(`^[\+]?[\d\s\-\(\)]+$`)
	return phoneRegex.MatchString(phone) && len(phone) >= 7
}

// Normalize methods
func (r *CreateUserRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Name = strings.TrimSpace(r.Name)
	r.Phone = strings.TrimSpace(r.Phone)
	if r.Role == "" {
		r.Role = RoleRider // Default role
	}
}

func (r *LoginRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

// ToUserInfo converts User to UserInfo (without sensitive data)
func (u *User) ToUserInfo() *UserInfo {
	return &UserInfo{
		ID:         u.ID,
		Email:      u.Email,
		Name:       u.Name,
		Phone:      u.Phone,
		Role:       u.Role,
		IsVerified: u.IsVerified,
	}
}