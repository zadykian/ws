package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/internal/conf"
)

// setting is a value for one of mac.conf's keys.
type setting struct{ key, value string }

// replace returns the text with the first line of key set to value and the later lines of key
// dropped, or with the line at its end where the text has none.
func replace(text, key, value string) string {
	var out []string
	done := false
	for _, l := range strings.SplitAfter(text, "\n") {
		if l == "" {
			continue
		}
		if name, _, ok := strings.Cut(strings.TrimLeft(l, " \t"), "="); ok && name == key {
			if !done {
				out = append(out, line(key, value)+"\n")
			}
			done = true
			continue
		}
		out = append(out, l)
	}
	if !done {
		if n := len(out); n > 0 && !strings.HasSuffix(out[n-1], "\n") {
			out[n-1] += "\n"
		}
		out = append(out, line(key, value)+"\n")
	}
	return strings.Join(out, "")
}

// change returns mac.conf's text, and its text with the settings, where the whole passes conf's
// checks and mac.conf is no link. They go through as one, as a pair of keys passes together alone.
func (f confFile) change(settings ...setting) (before, after string, err error) {
	before, err = f.read()
	if err != nil {
		return "", "", err
	}
	after = before
	lines := make([]string, 0, len(settings))
	for _, s := range settings {
		after = replace(after, s.key, s.value)
		lines = append(lines, line(s.key, s.value))
	}
	if _, errs := conf.Parse(after); len(errs) > 0 {
		for _, e := range errs {
			f.out.Say("%s", e.In(f.path))
		}
		return "", "", errRefused
	}
	if info, err := f.m.Stat(f.path); err == nil && info.Link != "" {
		f.out.Say("%s: %s is a link to %s, which the wizard leaves: set %s in its file, then run "+
			"the wizard again", f.stage, f.path, info.Link, strings.Join(lines, " "))
		return "", "", errRefused
	}
	return before, after, nil
}

// set gives key the value in mac.conf, as change checks.
func (f confFile) set(ctx context.Context, key, value string) error {
	return f.setAll(ctx, setting{key, value})
}

// setAll writes the settings in mac.conf, as change checks.
func (f confFile) setAll(ctx context.Context, settings ...setting) error {
	_, after, err := f.change(settings...)
	if err != nil {
		return err
	}
	return f.write(ctx, after)
}

// show prints the diff that set would write, under --check.
func (f confFile) show(key, value string) error {
	before, after, err := f.change(setting{key, value})
	if err != nil {
		return err
	}
	f.out.Diff(f.path, []byte(before), []byte(after))
	return nil
}
