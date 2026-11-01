// Package conf reads ~/.config/ws/mac.conf, the Mac's settings, without running it. It reads no
// file: the caller passes the text.
package conf

import (
	"fmt"
	"strings"
)

// Parse reads the text of mac.conf as Lines does, and where every line passes, returns the first
// refusal of the keys that go together.
func Parse(text string) (Conf, []Error) {
	c, errs := Lines(text)
	if len(errs) == 0 {
		errs = pairs(c)
	}
	return c, errs
}

// Lines reads the text of mac.conf, each line alone: a KEY=value a line, the value bare or in a
// pair of quotes, blank lines and lines starting with # skipped. It returns the error of every
// other line, of any other key and of a value of the wrong form. An empty value, KEY= or KEY="", is
// the key's default, as a missing line is. A later line of a key replaces an earlier one.
func Lines(text string) (Conf, []Error) {
	c := Conf{User: "root", Target: "lan"}
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
	return c, errs
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

// unquote takes off a pair of quotes, " or ', around raw, and says whether there was one.
func unquote(raw string) (string, bool) {
	if len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') && raw[len(raw)-1] == raw[0] {
		return raw[1 : len(raw)-1], true
	}
	return raw, false
}

// defaults sets the keys with a default other than empty to it.
func (c *Conf) defaults() {
	if c.User == "" {
		c.User = "root"
	}
	if c.Target == "" {
		c.Target = "lan"
	}
}
