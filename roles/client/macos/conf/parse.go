// Package conf reads ~/.config/ws/mac.conf, the Mac's settings, without running it. Parse and Lines
// take its text; File and Read reach the file through a machine.
package conf

import (
	"fmt"
	"strings"
)

// Parse reads the text of mac.conf as Lines does, and where every line passes, returns the first
// refusal of the keys that go together.
func Parse(text string) (Conf, []Error) { return Both.Parse(text) }

// Parse reads the text as Lines does, and where every line passes, returns the first refusal of
// the keys that go together.
func (s Scope) Parse(text string) (Conf, []Error) {
	c, errs := s.Lines(text)
	if len(errs) == 0 {
		errs = pairs(c)
	}
	return c, errs
}

// Lines reads the text of mac.conf, each line alone: a KEY=value a line, the value bare or in a
// pair of quotes. It skips blank lines and lines starting with #. It returns the error of every
// other line, of any other key and of a value of the wrong form. An empty value, KEY= or KEY="", is
// the key's default, as a missing line is. A later line of a key replaces an earlier one.
func Lines(text string) (Conf, []Error) { return Both.Lines(text) }

// Lines reads the text as the package's Lines does, and refuses a key of the other file as well.
func (s Scope) Lines(text string) (Conf, []Error) {
	c := Conf{Target: "lan"}
	var errs []Error
	for i, line := range strings.Split(text, "\n") {
		line = strings.Trim(line, " \t")
		if line == "" || line[0] == '#' {
			continue
		}
		name, raw, found := strings.Cut(line, "=")
		if !found || name == "" || name[0] < 'A' || name[0] > 'Z' {
			errs = append(errs, Error{i + 1, "not a KEY=value line"})
			continue
		}
		k, ok := lookup(name)
		if !ok {
			errs = append(errs, Error{i + 1, "unknown key " + name})
			continue
		}
		if msg := s.refuse(name); msg != "" {
			errs = append(errs, Error{i + 1, msg})
			continue
		}
		value, quoted := unquote(raw)
		if value != "" && !k.ok(value, quoted) {
			errs = append(errs, Error{i + 1, fmt.Sprintf("%s=%s is not %s", name, value, k.form)})
			continue
		}
		*k.field(&c) = value
		if value == "" {
			c.defaults()
		}
	}
	// The user's default follows the target, so it waits for the last line, whatever the order.
	if c.User == "" {
		c.User = defaultUser(c.Target)
	}
	return c, errs
}

// defaultUser is WS_USER's default: root on a cloud VM, which may have no other user, else the user
// that ws sets a machine up for.
func defaultUser(target string) string {
	if target == "cloud" {
		return "root"
	}
	return "zadykian"
}

// lookup finds the key named name.
func lookup(name string) (key, bool) {
	for _, k := range keys {
		if k.name == name {
			return k, true
		}
	}
	return key{}, false
}

// Value returns the value of the key named name, as the file gives it or by the key's default. It
// returns nothing for a name that mac.conf has no key of.
func (c Conf) Value(name string) string {
	k, ok := lookup(name)
	if !ok {
		return ""
	}
	return *k.field(&c)
}

// unquote takes off a pair of quotes, " or ', around raw, and says whether there was one.
func unquote(raw string) (string, bool) {
	if len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') && raw[len(raw)-1] == raw[0] {
		return raw[1 : len(raw)-1], true
	}
	return raw, false
}

// defaults sets the keys with a default other than empty to it, but WS_USER, which Lines sets last.
func (c *Conf) defaults() {
	if c.Target == "" {
		c.Target = "lan"
	}
}
