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
	// A line of base64, which apt skips where it starts with =, and the checksum's: an = and three
	// octets of base64 (sections 6 and 6.1).
	armorBase64   = regexp.MustCompile(`^(?:[A-Za-z0-9+/]+={0,2})?$`)
	armorChecksum = regexp.MustCompile(`^=[A-Za-z0-9+/]{4}$`)
)

// blanks are what apt strips from each line of a .asc key, C's isspace.
const blanks = " \t\n\v\f\r"

var errArmor = errors.New("not armored public key blocks alone")

// dearmor returns the data of armored, which must hold armored public key blocks alone, blank
// lines apart at most. apt dearmors a .asc key itself, and reads every block in it. It joins the
// lines after each block's headers into one run of base64, but those that start with = or -. Then
// it decodes the run in groups of four, and skips what is not base64. So each line here is base64
// that starts with no =, and each block decodes on its own, whole groups alone. The runs join as
// apt's does, and their groups end where the blocks do, so the data is the same as apt's.
func dearmor(armored string) ([]byte, error) {
	lines := strings.Split(strings.Trim(armored, blanks), "\n")
	for i, l := range lines {
		lines[i] = strings.Trim(l, blanks)
	}
	var data []byte
	for len(lines) > 0 {
		end := slices.Index(lines, armorEnd)
		if end < 0 {
			return nil, errArmor
		}
		block, err := dearmorBlock(lines[:end+1])
		if err != nil {
			return nil, err
		}
		data = append(data, block...)
		lines = lines[end+1:]
		for len(lines) > 0 && lines[0] == "" {
			lines = lines[1:]
		}
	}
	return data, nil
}

// dearmorBlock returns the data of one block's lines, from its first line to its last.
func dearmorBlock(lines []string) ([]byte, error) {
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
	if lines[0] != armorBegin || !all(headers, armorHeader) || !all(body, armorBase64) {
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
