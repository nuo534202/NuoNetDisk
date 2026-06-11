package model

import (
	"errors"
	"testing"
)

func TestErrorValues(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrNotFound", ErrNotFound, "NOT_FOUND"},
		{"ErrDuplicate", ErrDuplicate, "DUPLICATE"},
		{"ErrConflict", ErrConflict, "CONFLICT"},
		{"ErrForbidden", ErrForbidden, "FORBIDDEN"},
		{"ErrInvalidInput", ErrInvalidInput, "INVALID_INPUT"},
		{"ErrFileTooLarge", ErrFileTooLarge, "FILE_TOO_LARGE"},
		{"ErrRateLimited", ErrRateLimited, "RATE_LIMITED"},
		{"ErrTokenExpired", ErrTokenExpired, "TOKEN_EXPIRED"},
		{"ErrTokenRevoked", ErrTokenRevoked, "TOKEN_REVOKED"},
		{"ErrUnauthenticated", ErrUnauthenticated, "UNAUTHENTICATED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("expected %q, got %q", tt.msg, tt.err.Error())
			}
		})
	}
}

func TestErrorComparisons(t *testing.T) {
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Error("ErrNotFound should match itself")
	}
	if errors.Is(ErrNotFound, ErrDuplicate) {
		t.Error("ErrNotFound should not match ErrDuplicate")
	}
}
