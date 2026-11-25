package httputil

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorCode defines stable machine-readable API error codes.
type ErrorCode string

const (
	CodeUnauthorized    ErrorCode = "UNAUTHORIZED"
	CodeForbidden       ErrorCode = "FORBIDDEN"
	CodeTenantRequired  ErrorCode = "TENANT_REQUIRED"
	CodeValidationError ErrorCode = "VALIDATION_ERROR"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeConflict        ErrorCode = "CONFLICT"
	CodeInternalError   ErrorCode = "INTERNAL_ERROR"
)

// ErrorEnvelope is the consistent REST error response payload.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody carries the machine-readable code and human-readable message.
type ErrorBody struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// WriteError sends a consistent JSON error envelope.
func WriteError(c echo.Context, status int, code ErrorCode, message string) error {
	return c.JSON(status, ErrorEnvelope{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

// WriteBadRequest sends a 400 validation error envelope.
func WriteBadRequest(c echo.Context, message string) error {
	return WriteError(c, http.StatusBadRequest, CodeValidationError, message)
}
