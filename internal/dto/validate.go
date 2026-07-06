package dto

import (
	"net/mail"

	"github.com/zhuk/team-task-service/internal/apperr"
)

func validateEmail(email string) *apperr.Error {
	if email == "" {
		return apperr.Validation("email", "is required")
	}
	if len(email) > 255 {
		return apperr.Validation("email", "must be at most 255 characters")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return apperr.Validation("email", "must be a valid email address")
	}
	return nil
}
