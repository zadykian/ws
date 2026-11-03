package jetbrains

import (
	"fmt"
	"regexp"
	"strings"
)

var sha256Form = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ParseSum returns the SHA-256 that a checksum file holds: the file is the sum, then the archive's
// name. link names the file in the error.
func ParseSum(content []byte, link string) (string, error) {
	fields := strings.Fields(string(content))
	if len(fields) == 0 || !sha256Form.MatchString(fields[0]) {
		return "", fmt.Errorf("%s holds no SHA-256", link)
	}
	return fields[0], nil
}
