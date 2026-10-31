package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/mac/internal/conf"
)

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
				out = append(out, key+"="+value+"\n")
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
		out = append(out, key+"="+value+"\n")
	}
	return strings.Join(out, "")
}

// set gives key the value in mac.conf, where the whole passes conf's checks and mac.conf is no link.
func (f confFile) set(ctx context.Context, key, value string) error {
	before, err := f.read()
	if err != nil {
		return err
	}
	if _, errs := conf.Parse(replace(before, key, value)); len(errs) > 0 {
		for _, e := range errs {
			f.out.Say("%s", e.In(f.path))
		}
		return errRefused
	}
	if info, err := f.m.Stat(f.path); err == nil && info.Link != "" {
		f.out.Say("%s: %s is a link to %s, which the wizard leaves: set %s=%s in its file, then run "+
			"the wizard again", f.stage, f.path, info.Link, key, value)
		return errRefused
	}
	return f.write(ctx, replace(before, key, value))
}
