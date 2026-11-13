package launchd

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// Found returns, in order, the names of the servers with a trace of a daemon on the Mac. A trace is
// a plist, a state file, a log, or a directory under /etc/ws-tunnel. Found inverts For. Root lists
// that directory, which is root's. A name holds no dot, so private.key and the files of an older
// form fall out.
func Found(ctx context.Context, m effect.Machine) ([]string, error) {
	names, err := Installed(m)
	if err != nil {
		return nil, err
	}
	for _, from := range []struct{ dir, prefix, suffix string }{
		{"/var/run", "ws-tunnel.", ".state"},
		{"/var/log", "ws-tunnel.", ".log"},
	} {
		entries, err := m.ReadDir(from.dir)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		for _, e := range entries {
			if name, ok := nameIn(e, from.prefix, from.suffix); ok {
				names = append(names, name)
			}
		}
	}
	if _, err := m.RootAttrs(Dir); err == nil {
		out, err := m.RunRoot(ctx, "ls", "-1", Dir)
		if err != nil {
			return nil, err
		}
		for e := range strings.Lines(string(out)) {
			if e = strings.TrimSuffix(e, "\n"); e != "" && !strings.Contains(e, ".") {
				names = append(names, e)
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}
