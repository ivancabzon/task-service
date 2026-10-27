package apperr

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation error")
	ErrInternal     = errors.New("internal error")
)

type AppError struct {
	Code    string
	Message string // человекочитаемое сообщение
	Err     error  // sentinel / внутренняя
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func New(code, message string, sentinel error) *AppError {
	return &AppError{Code: code, Message: message, Err: sentinel}
}

func NotFound(msg string) *AppError     { return New("not_found", msg, ErrNotFound) }
func Forbidden(msg string) *AppError    { return New("forbidden", msg, ErrForbidden) }
func Unauthorized(msg string) *AppError { return New("unauthorized", msg, ErrUnauthorized) }
func Conflict(msg string) *AppError     { return New("conflict", msg, ErrConflict) }
func Validation(msg string) *AppError   { return New("validation_error", msg, ErrValidation) }
func Internal(msg string, err error) *AppError {
	return New("internal", msg, errors.Join(ErrInternal, err))
}
