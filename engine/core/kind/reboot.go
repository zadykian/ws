package kind

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
)

// moduleName is a kernel module's name, as /sys/module lists it: no path.
var moduleName = regexp.MustCompile(`^[a-z0-9_]+$`)

// rebootState notes a reboot that a package's module needs: a module that the kernel loads at
// boot alone, as a driver that a running display holds. While the module's version in force is not
// the package's, each run says so, with Hint after. It plans nothing.
type rebootState struct {
	Module  string `toml:"module"`
	Package string `toml:"package"`
	Hint    string `toml:"hint"`
}

func (s rebootState) resource(e Env) (resource.Resource, error) {
	if !moduleName.MatchString(s.Module) {
		return nil, fmt.Errorf("module %q is no kernel module's name", s.Module)
	}
	if !aptName.MatchString(s.Package) || strings.Contains(s.Package, "=") {
		return nil, fmt.Errorf("package %q is no package's name", s.Package)
	}
	return reboot{m: e.M, out: e.Out, s: s}, nil
}

// reboot is a reboot state's resource.
type reboot struct {
	m   effect.Machine
	out effect.Out
	s   rebootState
}

// Plan reads the module's version in force and dpkg's of the package. A package not
// installed, as in a dry run before its install, says nothing.
func (r reboot) Plan(ctx context.Context) ([]resource.Change, error) {
	states, err := apt.PackageStates(ctx, r.m, r.s.Package)
	if err != nil || !states[r.s.Package].Installed() {
		return nil, err
	}
	held := upstream(states[r.s.Package].Version)
	loaded, err := r.m.ReadFile("/sys/module/" + r.s.Module + "/version")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if running := cmp.Or(strings.TrimSpace(string(loaded)), "none"); running != held {
		note := fmt.Sprintf("the kernel runs the module %s at %s, and %s holds %s. Reboot to load it.",
			r.s.Module, running, r.s.Package, held)
		// Warn, as a write plans again after it applies, and the lines say a warning once.
		r.out.Warn("%s", strings.TrimSpace(note+" "+r.s.Hint))
	}
	return nil, nil
}

// upstream is the upstream part of a Debian version, as a module gives its own: the epoch N: and
// the revision after the last dash dropped.
func upstream(version string) string {
	if _, rest, ok := strings.Cut(version, ":"); ok {
		version = rest
	}
	if i := strings.LastIndex(version, "-"); i >= 0 {
		version = version[:i]
	}
	return version
}
