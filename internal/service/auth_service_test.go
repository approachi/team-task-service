package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/platform/jwtauth"
	"github.com/zhuk/team-task-service/internal/service"
)

func TestAuthService_RegisterAndLogin(t *testing.T) {
	repo := newFakeUserRepo()
	svc := service.NewAuthService(repo, []byte("test-secret"), time.Hour)
	ctx := context.Background()

	u, err := svc.Register(ctx, dto.RegisterRequest{Email: "a@example.com", Password: "password123", Name: "Alice"})
	require.NoError(t, err)
	require.Equal(t, "a@example.com", u.Email)
	require.NotEqual(t, "password123", u.PasswordHash)

	loggedIn, token, expiresAt, err := svc.Login(ctx, dto.LoginRequest{Email: "a@example.com", Password: "password123"})
	require.NoError(t, err)
	require.Equal(t, u.ID, loggedIn.ID)
	require.NotEmpty(t, token)
	require.True(t, expiresAt.After(time.Now()))

	claims, err := jwtauth.Parse([]byte("test-secret"), token)
	require.NoError(t, err)
	require.Equal(t, u.ID, claims.UserID)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	svc := service.NewAuthService(repo, []byte("test-secret"), time.Hour)
	ctx := context.Background()

	_, err := svc.Register(ctx, dto.RegisterRequest{Email: "a@example.com", Password: "password123", Name: "Alice"})
	require.NoError(t, err)

	_, _, _, err = svc.Login(ctx, dto.LoginRequest{Email: "a@example.com", Password: "wrongpass"})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeUnauthorized, appErr.Code)
}

func TestAuthService_Login_UnknownEmailSameErrorAsWrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	svc := service.NewAuthService(repo, []byte("test-secret"), time.Hour)

	_, _, _, err := svc.Login(context.Background(), dto.LoginRequest{Email: "nobody@example.com", Password: "password123"})
	require.Error(t, err)

	var appErr *apperr.Error
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, apperr.CodeUnauthorized, appErr.Code)
}

func TestPasswordHashing_Roundtrip(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte("password123")))
	require.Error(t, bcrypt.CompareHashAndPassword(hash, []byte("wrongpass")))
}

func TestJWT_IssueParseRoundtrip(t *testing.T) {
	secret := []byte("test-secret")
	token, expiresAt, err := jwtauth.Issue(secret, time.Hour, 42, "a@example.com")
	require.NoError(t, err)
	require.True(t, expiresAt.After(time.Now()))

	claims, err := jwtauth.Parse(secret, token)
	require.NoError(t, err)
	require.Equal(t, int64(42), claims.UserID)
	require.Equal(t, "a@example.com", claims.Email)
}

func TestJWT_ExpiredTokenRejected(t *testing.T) {
	secret := []byte("test-secret")
	token, _, err := jwtauth.Issue(secret, -time.Hour, 1, "a@example.com")
	require.NoError(t, err)

	_, err = jwtauth.Parse(secret, token)
	require.Error(t, err)
}

func TestJWT_TamperedSignatureRejected(t *testing.T) {
	token, _, err := jwtauth.Issue([]byte("real-secret"), time.Hour, 1, "a@example.com")
	require.NoError(t, err)

	_, err = jwtauth.Parse([]byte("wrong-secret"), token)
	require.Error(t, err)
}
