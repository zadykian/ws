// Package check holds the pure checks of the forms that mac.conf and tunnel.conf take: an address,
// a network, a name, a port and a key.
package check

import (
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// IPv4 reports whether s is an IPv4 address in four decimal parts, none with a leading zero.
func IPv4(s string) bool {
	a, err := netip.ParseAddr(s)
	return err == nil && a.Is4()
}

// CIDR takes an IPv4 network with no bits set past its length, as a route needs.
func CIDR(s string) (netip.Prefix, bool) {
	p, err := netip.ParsePrefix(s)
	if err != nil || !p.Addr().Is4() || p != p.Masked() {
		return netip.Prefix{}, false
	}
	return p, true
}

// Name reports whether s is DNS labels joined by dots, of 63 characters at most each, and 253 in
// all.
func Name(s string) bool {
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

// Port reports whether s is 1 to 65535, with no leading zero or sign.
func Port(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1 && n <= 65535 && s == strconv.Itoa(n)
}

var keyForm = regexp.MustCompile(`^[A-Za-z0-9+/]{43}=$`)

// Key reports whether s is a WireGuard key in base64: 43 characters and an =.
func Key(s string) bool {
	return keyForm.MatchString(s)
}
