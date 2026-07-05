package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/model"
	"github.com/zhuk/team-task-service/internal/platform/jwtauth"
)

type AuthService struct {
	users     UserRepository
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewAuthService(users UserRepository, jwtSecret []byte, jwtTTL time.Duration) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

// TODO: добавить подтверждение email при регистрации и восстановление
// пароля по ссылке — сейчас любой email принимается без верификации.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &model.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
	}
	return s.users.Create(ctx, u)
}

// Login returns the authenticated user, a signed JWT, and its expiry. On any
// credential failure it returns the same apperr.Unauthorized message
// regardless of whether the email or the password was wrong, so the
// response never reveals which one.
//
// TODO: токен нельзя отозвать до истечения TTL — нет logout/blacklist и
// нет refresh-токенов, поэтому скомпрометированный или "отозванный" токен
// остаётся валидным до 24ч. Для прода — короткий access-токен (5-15 мин) +
// refresh-токен с ротацией и хранением в Redis (allowlist по jti).
//
// TODO: нет блокировки после N неудачных попыток входа — уязвимо к
// brute-force подбору пароля даже с rate-limit'ом (см. TODO в
// internal/platform/middleware/ratelimit.go про IP-лимит на /login).
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*model.User, string, time.Time, error) {
	u, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		var appErr *apperr.Error
		if errors.As(err, &appErr) && appErr.Code == apperr.CodeNotFound {
			return nil, "", time.Time{}, apperr.Unauthorized("invalid email or password")
		}
		return nil, "", time.Time{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", time.Time{}, apperr.Unauthorized("invalid email or password")
	}

	token, expiresAt, err := jwtauth.Issue(s.jwtSecret, s.jwtTTL, u.ID, u.Email)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("issue token: %w", err)
	}

	return u, token, expiresAt, nil
}
