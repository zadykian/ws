// Package values reads each server's values.yml, in /etc/ws/servers, as YAML without running it. Parse
// and Lines take a file's text; ReadFleet reaches the files through a machine.
package values

import "fmt"

// Parse reads the text of a server's file as Lines does, and where every line passes, returns the
// first refusal of the keys that go together.
func Parse(text string) (Conf, []Error) {
	c, errs := Lines(text)
	if len(errs) == 0 {
		errs = pairs(c)
	}
	return c, errs
}

// Lines reads the text of a server's file: YAML, a KEY: value a line. It reports an error for a key
// that is unknown, a key given twice, a value that is no scalar, and a value of the wrong form. An
// empty value, KEY: or KEY: "", is the key's default, as a missing line is.
func Lines(text string) (Conf, []Error) {
	c := Conf{Target: "lan", Slot: "0"}
	pairs, errs := entries(text)
	for _, e := range pairs {
		k, ok := lookup(e.key)
		if !ok {
			errs = append(errs, Error{e.line, "unknown key " + e.key})
			continue
		}
		if e.err != "" {
			errs = append(errs, Error{e.line, e.err})
			continue
		}
		if e.value != "" && !k.ok(e.value) {
			errs = append(errs, Error{e.line, fmt.Sprintf("%s=%s is not %s", e.key, e.value, k.form)})
			continue
		}
		*k.field(&c) = e.value
		if e.value == "" {
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

// Value returns the value of the key named name, as the file gives it or by the key's default. It
// returns nothing for a name that is no key.
func (c Conf) Value(name string) string {
	k, ok := lookup(name)
	if !ok {
		return ""
	}
	return *k.field(&c)
}

// defaults sets the keys with a default other than empty to it (the target and the slot).
func (c *Conf) defaults() {
	if c.Target == "" {
		c.Target = "lan"
	}
	if c.Slot == "" {
		c.Slot = "0"
	}
}
