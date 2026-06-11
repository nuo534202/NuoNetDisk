package model

import "errors"

var (
	ErrNotFound        = errors.New("NOT_FOUND")
	ErrDuplicate       = errors.New("DUPLICATE")
	ErrConflict        = errors.New("CONFLICT")
	ErrForbidden       = errors.New("FORBIDDEN")
	ErrInvalidInput    = errors.New("INVALID_INPUT")
	ErrFileTooLarge    = errors.New("FILE_TOO_LARGE")
	ErrRateLimited     = errors.New("RATE_LIMITED")
	ErrTokenExpired    = errors.New("TOKEN_EXPIRED")
	ErrTokenRevoked    = errors.New("TOKEN_REVOKED")
	ErrUnauthenticated = errors.New("UNAUTHENTICATED")
)
