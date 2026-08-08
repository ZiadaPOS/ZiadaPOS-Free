// Package apierr defines a structured error used by services to signal the
// HTTP handler exactly how to respond. Errors created here are safe to
// translate into the response envelope.
package apierr

import (
	"errors"
	"net/http"

	"gorm.io/gorm"
)

// Error is a domain error with an HTTP status and machine code.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Message }

// New builds a domain error.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// WithField attaches a field-level validation message.
func (e *Error) WithField(field, msg string) *Error {
	if e.Fields == nil {
		e.Fields = map[string]string{}
	}
	e.Fields[field] = msg
	return e
}

// Convenience constructors.
func BadRequest(msg string) *Error          { return New(http.StatusBadRequest, "validation_error", msg) }
func Unauthorized(msg string) *Error        { return New(http.StatusUnauthorized, "unauthorized", msg) }
func Forbidden(msg string) *Error           { return New(http.StatusForbidden, "forbidden", msg) }
func NotFound(msg string) *Error            { return New(http.StatusNotFound, "not_found", msg) }
func Conflict(msg string) *Error            { return New(http.StatusConflict, "conflict", msg) }
func Locked(msg string) *Error              { return New(http.StatusLocked, "account_locked", msg) }
func RateLimited(msg string) *Error         { return New(http.StatusTooManyRequests, "rate_limited", msg) }
func Internal(msg string) *Error            { return New(http.StatusInternalServerError, "internal_error", msg) }

// Translate converts any error into an *Error. GORM not-found becomes a 404;
// unexpected errors become 500s.
func Translate(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return NotFound("Resource not found")
	}
	return Internal(err.Error())
}
