// Package tunnel holds the pure part of ws tunnel, the Mac's WireGuard daemon: the settings and
// their checks, the choice of a way to ws, the routes, and the state that status prints. It reads
// no file and runs no command, and takes the time as a parameter.
package tunnel

import (
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// isIPv4 reports whether s is an IPv4 address in four decimal parts, none with a leading zero.
func isIPv4(s string) bool {
	a, err := netip.ParseAddr(s)
	return err == nil && a.Is4()
}

// parseCIDR takes an IPv4 network with no bits set past its length, as a route needs.
func parseCIDR(s string) (netip.Prefix, bool) {
	p, err := netip.ParsePrefix(s)
	if err != nil || !p.Addr().Is4() || p != p.Masked() {
		return netip.Prefix{}, false
	}
	return p, true
}

// isName reports whether s is DNS labels joined by dots, of 63 characters at most each, and 253 in
// all.
func isName(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if c != '-' && (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
				return false
			}
		}
	}
	return true
}

// isPort reports whether s is 1 to 65535, with no leading zero or sign.
func isPort(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1 && n <= 65535 && s == strconv.Itoa(n)
}

var keyForm = regexp.MustCompile(`^[A-Za-z0-9+/]{43}=$`)

// isKey reports whether s is a WireGuard key in base64: 43 characters and an =.
func isKey(s string) bool {
	return keyForm.MatchString(s)
}
