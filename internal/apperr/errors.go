// Package apperr defines the typed application error used across the service
// and its mapping to an HTTP status code + machine-readable code, so every
// handler produces the same error envelope shape without repeating the logic.
package apperr

import (
	"errors"
	"net/http"
)

type Code string

const (
	CodeValidation      Code = "VALIDATION_ERROR"
	CodeUnauthorized    Code = "UNAUTHORIZED"
	CodeForbidden       Code = "FORBIDDEN"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeTooManyRequests Code = "TOO_MANY_REQUESTS"
	CodeInternal        Code = "INTERNAL"
)

type Error struct {
	Code    Code
	Message string
	Details map[string]string
	Err     error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func Validation(field, message string) *Error {
	return &Error{
		Code:    CodeValidation,
		Message: "validation failed",
		Details: map[string]string{field: message},
	}
}

func Unauthorized(message string) *Error {
	return &Error{Code: CodeUnauthorized, Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Code: CodeForbidden, Message: message}
}

func NotFound(message string) *Error {
	return &Error{Code: CodeNotFound, Message: message}
}

func Conflict(message string) *Error {
	return &Error{Code: CodeConflict, Message: message}
}

func TooManyRequests(message string) *Error {
	return &Error{Code: CodeTooManyRequests, Message: message}
}

func Internal(err error) *Error {
	return &Error{Code: CodeInternal, Message: "internal server error", Err: err}
}

// StatusFor maps a Code to its HTTP status. Unknown codes map to 500 so a
// missed mapping fails safe rather than leaking a 200 or a panic.
func StatusFor(code Code) int {
	switch code {
	case CodeValidation:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeTooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// As extracts an *Error from err, wrapping it as an internal error if it
// isn't already a typed *Error (e.g. an unexpected driver/db error).
func As(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if ok := errors.As(err, &appErr); ok {
		return appErr
	}
	return Internal(err)
}
