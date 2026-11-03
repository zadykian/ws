package fans

import (
	"errors"
	"fmt"
)

// MissingError is a device that cannot be read or set now. The daemon tries it again on its next
// tick: a driver may load after the service starts, or come back after an update.
type MissingError struct{ Msg string }

func (e *MissingError) Error() string { return e.Msg }

// missing returns a MissingError of the formatted message.
func missing(format string, args ...any) error {
	return &MissingError{fmt.Sprintf(format, args...)}
}

// IsMissing reports whether err is, or wraps, a MissingError.
func IsMissing(err error) bool {
	var m *MissingError
	return errors.As(err, &m)
}
