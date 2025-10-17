package httpserver

import (
	"strings"

	"github.com/charmbracelet/log"
)

func validateCreateUserRequest(req *CreateUserRequest, log *log.Logger) error {
	if req.Name == "" {
		return NewValidationError("Name is required")
	}

	if len(req.Name) < 2 {
		return NewValidationError("Name must be at least 2 characters long")
	}

	if len(req.Name) > 28 {
		return NewValidationError("Name must be not more that 28 characters long")
	}

	if req.Email == "" {
		return NewValidationError("Email is required")
	}

	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		return NewValidationError("Invalid email format")
	}

	if err := validatePassword(req.Password); err != nil {
		return err
	}

	return nil
}

func validatePassword(pw string) error {
	if len(pw) < 8 {
		return NewValidationError("Password must be at least 8 characters")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, c := range pw {
		switch {
		case 'A' <= c && c <= 'Z':
			hasUpper = true
		case 'a' <= c && c <= 'z':
			hasLower = true
		case '0' <= c && c <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*", c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return NewValidationError("Password must contain an uppercase letter")
	}
	if !hasLower {
		return NewValidationError("Password must contain a lowercase letter")
	}
	if !hasDigit {
		return NewValidationError("Password must contain a number")
	}
	if !hasSpecial {
		return NewValidationError("Password must contain a special character")
	}

	return nil
}
