package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/platform/middleware"
)

func parseIDParam(r *http.Request, name string) (int64, error) {
	raw := chi.URLParam(r, name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperr.Validation(name, "must be a positive integer")
	}
	return id, nil
}

func requireUserID(r *http.Request) (int64, error) {
	id, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		return 0, apperr.Unauthorized("missing authentication")
	}
	return id, nil
}
