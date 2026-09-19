package model

import "fmt"

const (
	ErrCodeInvalidArgument = "INVALID_ARGUMENT"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeConflict        = "CONFLICT"
)

type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func invalid(msg string) *ValidationError {
	return &ValidationError{Code: ErrCodeInvalidArgument, Message: msg}
}

func invalidf(format string, args ...any) *ValidationError {
	return invalid(fmt.Sprintf(format, args...))
}

type AuthError struct {
	Code    string
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

func unauthorized(msg string) *AuthError {
	return &AuthError{Code: ErrCodeUnauthorized, Message: msg}
}

type NotFoundError struct {
	Code    string
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

func notFound(msg string) *NotFoundError {
	return &NotFoundError{Code: ErrCodeNotFound, Message: msg}
}

type ConflictError struct {
	Code    string
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

func conflict(msg string) *ConflictError {
	return &ConflictError{Code: ErrCodeConflict, Message: msg}
}

func Unauthorized(msg string) *AuthError {
	return unauthorized(msg)
}

func NotFound(msg string) *NotFoundError {
	return notFound(msg)
}

func Conflict(msg string) *ConflictError {
	return conflict(msg)
}
