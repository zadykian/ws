package kind

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// System is the operating system that a side's units run on, as the tools that find a home differ.
// The side sets it in Env, and Linux is the zero value.
type System int

const (
	// Linux finds a home in the passwd database, by getent.
	Linux System = iota
	// MacOS finds a home in Directory Service, by dscl, as macOS has no getent.
	MacOS
)

// home is the home of name, an account other than root.
func (s System) home(ctx context.Context, m effect.Machine, name string) (string, error) {
	var out []byte
	var err error
	dir := ""
	switch s {
	case MacOS:
		out, err = m.Run(ctx, "dscl", ".", "-read", "/Users/"+name, "NFSHomeDirectory")
		// dscl puts a long value on a line of its own, below the key.
		if rest, ok := strings.CutPrefix(strings.TrimSpace(string(out)), "NFSHomeDirectory:"); ok {
			dir = strings.TrimSpace(rest)
		}
	default:
		out, err = m.Run(ctx, "getent", "passwd", name)
		if fields := strings.Split(strings.TrimSpace(string(out)), ":"); len(fields) == 7 {
			dir = fields[5]
		}
	}
	if err != nil {
		return "", fmt.Errorf("find the home of %s: %w", name, err)
	}
	if !path.IsAbs(dir) || path.Clean(dir) == "/" {
		return "", fmt.Errorf("find the home of %s: %q names no home", name, out)
	}
	return path.Clean(dir), nil
}
