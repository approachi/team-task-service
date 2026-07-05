// Package httpx holds the shared HTTP response envelope and pagination
// helpers used by every handler, so every endpoint returns the same
// success/error shape.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/zhuk/team-task-service/internal/apperr"
)

type Envelope struct {
	Data any   `json:"data,omitempty"`
	Meta *Meta `json:"meta,omitempty"`
}

type Meta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode json response", "error", err)
	}
}

func WriteData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, Envelope{Data: data})
}

func WriteList(w http.ResponseWriter, status int, data any, meta Meta) {
	WriteJSON(w, status, Envelope{Data: data, Meta: &meta})
}

func WriteError(w http.ResponseWriter, err error) {
	appErr := apperr.As(err)
	if appErr.Code == apperr.CodeInternal {
		slog.Error("internal error", "error", appErr.Err)
	}
	WriteJSON(w, apperr.StatusFor(appErr.Code), ErrorEnvelope{
		Error: ErrorBody{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}
