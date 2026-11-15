package values

import (
	"context"
	"strings"
)

// Setting is a value for one of values.yml's keys.
type Setting struct{ Key, Value string }

// replace returns the text with the first line of key set to value and the later lines of key
// dropped. Where the text has no line of key, it adds the line at its end.
func replace(text, key, value string) string {
	var out []string
	done := false
	for _, l := range strings.SplitAfter(text, "\n") {
		if l == "" {
			continue
		}
		if name, ok := lineKey(l); ok && name == key {
			if !done {
				out = append(out, Text(key, value))
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
		out = append(out, Text(key, value))
	}
	return strings.Join(out, "")
}

// change returns values.yml's text, and its text with the settings, where the whole passes the package's
// checks and values.yml is no link. They go through as one, as a pair of keys passes together alone.
func (f File) change(settings ...Setting) (before, after string, err error) {
	before, err = f.Read()
	if err != nil {
		return "", "", err
	}
	after = before
	lines := make([]string, 0, len(settings))
	for _, s := range settings {
		after = replace(after, s.Key, s.Value)
		lines = append(lines, Line(s.Key, s.Value))
	}
	if _, errs := f.Scope.Parse(after); len(errs) > 0 {
		for _, e := range errs {
			f.Out.Say("%s", e.In(f.Path))
		}
		return "", "", ErrRefused
	}
	if info, err := f.M.Stat(f.Path); err == nil && info.Link != "" {
		f.Out.Say("%s: %s is a link to %s, which the wizard leaves: set %s in its file, then run "+
			"the wizard again", f.Stage, f.Path, info.Link, strings.Join(lines, " "))
		return "", "", ErrRefused
	}
	return before, after, nil
}

// Set gives key the value in values.yml, as change checks.
func (f File) Set(ctx context.Context, key, value string) error {
	return f.SetAll(ctx, Setting{Key: key, Value: value})
}

// SetAll writes the settings in values.yml, as change checks.
func (f File) SetAll(ctx context.Context, settings ...Setting) error {
	_, after, err := f.change(settings...)
	if err != nil {
		return err
	}
	return f.Write(ctx, after)
}

// Show prints the diff that Set would write, under --check.
func (f File) Show(key, value string) error {
	before, after, err := f.change(Setting{Key: key, Value: value})
	if err != nil {
		return err
	}
	f.Out.Diff(f.Path, []byte(before), []byte(after))
	return nil
}
