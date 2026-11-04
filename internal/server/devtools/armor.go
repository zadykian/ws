// Package devtools sets up the tools a development server needs: Node, the .NET SDKs, gh, glow, the
// libraries headless Chromium needs, Go, gopls, Delve, roslyn-language-server and rustup.
package devtools

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// The armor's first and last lines, and a header, a key and its value (RFC 9580, section 6.2).
const (
	armorBegin = "-----BEGIN PGP PUBLIC KEY BLOCK-----"
	armorEnd   = "-----END PGP PUBLIC KEY BLOCK-----"
)

var (
	armorHeader = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]*:(?: .*)?$`)
	// A line of base64, and the checksum's: an = and three octets of base64 (sections 6 and 6.1).
	armorBase64   = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	armorChecksum = regexp.MustCompile(`^=[A-Za-z0-9+/]{4}$`)
)

// blanks are what apt strips from each line of a .asc key, C's isspace.
const blanks = " \t\n\v\f\r"

var errArmor = errors.New("not one armored public key alone")

// dearmor returns the data of armored, which must be one armored public key block alone. apt
// dearmors a .asc key itself, and reads on into any block that follows. It decodes each line after
// the headers but those that start with = or -, and skips what is not base64. So each line here
// must be base64, and the file must hold this block alone.
func dearmor(armored string) ([]byte, error) {
	lines := strings.Split(strings.Trim(armored, blanks), "\n")
	for i, l := range lines {
		lines[i] = strings.Trim(l, blanks)
	}
	blank := slices.Index(lines, "")
	if blank < 0 {
		blank = len(lines)
	}
	var headers, body []string
	if blank+1 <= len(lines)-1 {
		body = slices.Clone(lines[blank+1 : len(lines)-1])
	}
	if blank > 1 {
		headers = lines[1:blank]
	}
	if len(body) > 0 && armorChecksum.MatchString(body[len(body)-1]) {
		body = body[:len(body)-1]
	}
	if lines[0] != armorBegin || lines[len(lines)-1] != armorEnd ||
		!all(headers, armorHeader) || !all(body, armorBase64) {
		return nil, errArmor
	}
	data, err := base64.StdEncoding.DecodeString(strings.Join(body, ""))
	if err != nil {
		return nil, fmt.Errorf("the armor's base64: %w", err)
	}
	return data, nil
}

func all(lines []string, re *regexp.Regexp) bool {
	return !slices.ContainsFunc(lines, func(l string) bool { return !re.MatchString(l) })
}
