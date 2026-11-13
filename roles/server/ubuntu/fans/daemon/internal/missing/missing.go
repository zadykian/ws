// Package missing marks the errors of a device that cannot be read or set now.
package missing

import (
	"errors"
	"fmt"
)

// Error is a device that cannot be read or set now. The daemon tries it again on its next
// tick: a driver may load after the service starts, or come back after an update.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

// Errorf returns an Error of the formatted message.
func Errorf(format string, args ...any) error {
	return &Error{fmt.Sprintf(format, args...)}
}

// Is reports whether err is, or wraps, an Error.
func Is(err error) bool {
	var m *Error
	return errors.As(err, &m)
}
