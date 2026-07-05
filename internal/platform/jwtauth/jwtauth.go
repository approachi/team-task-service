// Package jwtauth issues and parses the service's own JWTs. It lives under
// platform (not service or handler) because both the auth service (issuing
// on login) and the auth middleware (parsing on every protected request)
// depend on it, and platform code must not depend upward on service code.
package jwtauth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "team-task-service"

type Claims struct {
	UserID int64  `json:"uid"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func Issue(secret []byte, ttl time.Duration, userID int64, email string) (token string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(ttl)

	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

func Parse(secret []byte, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}
	return claims, nil
}
