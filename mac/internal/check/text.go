package check

import (
	"strings"
	"unicode"
)

// only reports whether s holds no character but those of set.
func only(s, set string) bool {
	return strings.Trim(s, set) == ""
}

const (
	letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	digits  = "0123456789"
)

func hasControl(s string) bool {
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}

// CablePort reports whether s is a port of ws's, as enp5s0: up to 15 letters, digits and ._-, with
// no - first.
func CablePort(s string) bool {
	return len(s) <= 15 && !strings.HasPrefix(s, "-") && only(s, letters+digits+"_.-")
}

// CableService reports whether s is a network service as networksetup -listallnetworkservices
// prints it: no * first, a mark of a disabled service, and no control character. A blank needs
// quotes, as in a shell's assignment.
func CableService(s string, quoted bool) bool {
	if strings.HasPrefix(s, "*") || hasControl(s) {
		return false
	}
	return quoted || !strings.Contains(s, " ")
}

// GitName reports whether s is a name that git's config takes in quotes, which a " would end and a
// \ escape. A blank needs quotes, as in a shell's assignment.
func GitName(s string, quoted bool) bool {
	if hasControl(s) || strings.ContainsAny(s, `"\`) {
		return false
	}
	return quoted || !strings.Contains(s, " ")
}

// GitEmail reports whether s is an email address: letters, digits and ._%+- before the @, and a
// host name after it.
func GitEmail(s string) bool {
	local, host, found := strings.Cut(s, "@")
	return found && local != "" && only(local, letters+digits+"._%+-") && Name(host)
}

// HookSecret reports whether s is a Keychain item's name: 1 to 64 letters, digits and ._-.
func HookSecret(s string) bool {
	return s != "" && len(s) <= 64 && only(s, letters+digits+"._-")
}
