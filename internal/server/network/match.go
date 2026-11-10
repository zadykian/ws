package network

import (
	"fmt"
	"slices"
	"strings"
)

// Port is the cable's interface as the machine shows it.
type Port struct {
	Name     string
	Altnames []string
	// MAC is the permanent address.
	MAC string
	// Driver is udev's ID_NET_DRIVER, empty where udev names none.
	Driver string
}

// globCharacters in a match: name may make it fit more than one interface: networkd reads globs,
// and a leading "!" inverts the test.
const globCharacters = "*?[!"

// text returns a YAML scalar as Python's str showed it, or a list as one string.
func text(value any) string {
	return fmt.Sprint(value)
}

// globs returns the globs networkd reads from a match: value, which netplan writes as is.
// networkd splits the value at whitespace, and a "!" before the list inverts the test of each.
func globs(value any) []string {
	var whole string
	if list, ok := value.([]any); ok {
		parts := make([]string, len(list))
		for i, each := range list {
			parts[i] = text(each)
		}
		whole = strings.Join(parts, " ")
	} else {
		whole = text(value)
	}
	fields := strings.Fields(strings.TrimPrefix(whole, "!"))
	if strings.HasPrefix(whole, "!") {
		for i, glob := range fields {
			fields[i] = "!" + glob
		}
	}
	return fields
}

// passes tells whether value passes networkd's test of the globs, which fails where a "!" glob
// fits.
func passes(list []string, value string) bool {
	var wanted []string
	for _, glob := range list {
		if negated, ok := strings.CutPrefix(glob, "!"); ok {
			if globMatches(negated, value) {
				return false
			}
		} else {
			wanted = append(wanted, glob)
		}
	}
	return len(wanted) == 0 || slices.ContainsFunc(wanted, func(g string) bool { return globMatches(g, value) })
}

// fits tells whether netplan's match: fits the port, as networkd's [Match] tests it. A name fits
// the port's name or any of its alternative names, and macaddress the permanent address. A driver
// udev doesn't name, and a key netplan adds later, count as a fit. The check then refuses rather
// than misses an entry.
func fits(match map[string]any, port Port) bool {
	names := append([]string{port.Name}, port.Altnames...)
	for key, value := range match {
		list := globs(value)
		switch key {
		case "name":
			if !slices.ContainsFunc(names, func(n string) bool { return passes(list, n) }) {
				return false
			}
		case "macaddress":
			if !strings.EqualFold(text(value), port.MAC) {
				return false
			}
		case "driver":
			if port.Driver != "" && !passes(list, port.Driver) {
				return false
			}
		}
	}
	return true
}

// alone tells whether a match: that fits the port fits no other interface. A permanent MAC
// address is one interface's, and so is a name without a glob, as the kernel gives no two
// interfaces one name. A list of names holds a bracket, so it never counts.
func alone(match map[string]any) bool {
	_, mac := match["macaddress"]
	name, named := match["name"]
	_, list := name.([]any)
	return mac || (named && !list && !strings.ContainsAny(text(name), globCharacters))
}
