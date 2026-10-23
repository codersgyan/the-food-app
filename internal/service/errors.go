// Package service holds the app's business rules: who can do what, and when.
// It knows nothing about HTTP.
package service

import (
	"errors"
	"fmt"
)

// The kinds of mistake a client can make. The handler turns each kind into
// an HTTP status code.
var (
	ErrInvalid      = errors.New("invalid request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

// Error is a failure the client caused. Msg is safe to show to the client.
type Error struct {
	Kind error
	Msg  string
}

func (e *Error) Error() string { return e.Msg }
func (e *Error) Unwrap() error { return e.Kind }

func fail(kind error, format string, args ...any) error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}
