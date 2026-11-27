package values

import (
	"fmt"
	"strings"
)

// check refuses a port that the role's netplan file could not hold, where a port is set. The file
// quotes the port in YAML's double quotes. So check refuses each byte that would need an escape
// there, with the others that JSON escapes. Without a port, nothing is checked, as the role writes
// no file.
func (c Cable) check() error {
	if c.Port == "" {
		return nil
	}
	if strings.ContainsFunc(c.Port, func(r rune) bool {
		return r <= ' ' || r == 0x7f || strings.ContainsRune("/\"\\<>&", r)
	}) {
		return fmt.Errorf("WS_CABLE_PORT %q is not an interface's name", c.Port)
	}
	return nil
}
