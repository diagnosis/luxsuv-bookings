package utils

import (
	"strings"
	"unicode"
)

// NormalizeString trims whitespace and normalizes string input
func NormalizeString(s string) string {
	return strings.TrimSpace(s)
}

// NormalizeEmail normalizes email addresses (lowercase and trim)
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// NormalizePhone normalizes phone numbers (basic cleaning)
func NormalizePhone(phone string) string {
	// Remove all non-digit characters except + at the beginning
	cleaned := strings.TrimSpace(phone)
	if cleaned == "" {
		return ""
	}
	
	var result strings.Builder
	for i, r := range cleaned {
		if i == 0 && r == '+' {
			result.WriteRune(r)
		} else if unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// IsValidEmail performs basic email validation
func IsValidEmail(email string) bool {
	normalized := NormalizeEmail(email)
	if normalized == "" {
		return false
	}
	
	// Very basic email validation - contains @ and domain
	parts := strings.Split(normalized, "@")
	if len(parts) != 2 {
		return false
	}
	
	local, domain := parts[0], parts[1]
	return len(local) > 0 && len(domain) > 2 && strings.Contains(domain, ".")
}

// IsValidPhone performs basic phone validation
func IsValidPhone(phone string) bool {
	normalized := NormalizePhone(phone)
	if len(normalized) < 7 { // Minimum reasonable phone length
		return false
	}
	
	// Should start with + or digit
	first := rune(normalized[0])
	return first == '+' || unicode.IsDigit(first)
}