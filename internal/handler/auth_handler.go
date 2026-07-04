package handler

import (
	"net/http"

	"github.com/zhuk/team-task-service/internal/dto"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
	"github.com/zhuk/team-task-service/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Registration payload"
// @Success      201 {object} httpx.Envelope{data=dto.UserResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      409 {object} httpx.ErrorEnvelope
// @Router       /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	u, err := h.auth.Register(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteData(w, http.StatusCreated, dto.NewUserResponse(u))
}

// Login godoc
// @Summary      Log in and receive a JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Credentials"
// @Success      200 {object} httpx.Envelope{data=dto.LoginResponse}
// @Failure      400 {object} httpx.ErrorEnvelope
// @Failure      401 {object} httpx.ErrorEnvelope
// @Router       /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		httpx.WriteError(w, err)
		return
	}

	u, token, expiresAt, err := h.auth.Login(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteData(w, http.StatusOK, dto.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      dto.NewUserResponse(u),
	})
}
