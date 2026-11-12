package host

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/effect"
)

// dpkgState is what dpkg-query says of a package: its version and its status, such as ii, rc or
// iU, whole. A package dpkg doesn't know has the zero state.
type dpkgState struct {
	version, status string
}

// dpkgFormat is the one format of dpkg-query that ws reads.
const dpkgFormat = "${Package} ${Version} ${db:Status-Abbrev}\n"

// dpkgStates asks dpkg for the packages of patterns, names or globs, and no others, as a report
// covers what ws manages. No pattern runs nothing: dpkg-query lists every package for none.
func dpkgStates(ctx context.Context, m effect.Machine, patterns ...string) (map[string]dpkgState, error) {
	states := map[string]dpkgState{}
	if len(patterns) == 0 {
		return states, nil
	}
	args := append([]string{"-W", "-f", dpkgFormat}, patterns...)
	out, err := m.Run(ctx, "dpkg-query", args...)
	// dpkg-query exits with 1 for a package it does not know, and prints the others.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, err
	}
	for line := range strings.Lines(string(out)) {
		f := strings.Fields(line)
		if len(f) < 2 || len(f) > 3 {
			continue
		}
		// A line of two fields has no version: dpkg knows of a package it never installed.
		state := dpkgState{status: f[len(f)-1]}
		if len(f) == 3 {
			state.version = f[1]
		}
		// A package listed for two architectures keeps its installed line.
		if have, ok := states[f[0]]; !ok || !have.installed() {
			states[f[0]] = state
		}
	}
	return states, nil
}

// installed is true for ii alone: plans and reports count a package in any other state, such as
// iU or hi, as not installed.
func (s dpkgState) installed() bool { return s.status == "ii" }

// settingsLeft is true for a package that was removed and kept its settings.
func (s dpkgState) settingsLeft() bool { return len(s.status) >= 2 && s.status[1] == 'c' }

// present is true for a package with anything of it on the machine: the status's second letter
// is n for nothing.
func (s dpkgState) present() bool { return len(s.status) >= 2 && s.status[1] != 'n' }

func (s dpkgState) text() string {
	switch {
	case s.installed():
		return s.version
	case s.settingsLeft():
		return "settings left"
	case !s.present():
		return missing
	}
	return s.version + " (" + s.status + ")"
}
