package dto

import (
	"net/mail"
	"time"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/model"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (r RegisterRequest) Validate() *apperr.Error {
	if err := validateEmail(r.Email); err != nil {
		return err
	}
	if len(r.Password) < 8 || len(r.Password) > 72 {
		return apperr.Validation("password", "must be between 8 and 72 characters")
	}
	if r.Name == "" || len(r.Name) > 255 {
		return apperr.Validation("name", "is required and must be at most 255 characters")
	}
	return nil
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(u *model.User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() *apperr.Error {
	if err := validateEmail(r.Email); err != nil {
		return err
	}
	if r.Password == "" {
		return apperr.Validation("password", "is required")
	}
	return nil
}

type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

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
