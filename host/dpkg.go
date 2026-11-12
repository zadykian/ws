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

// dpkgStates asks dpkg for the packages of names and no others, as a report covers what ws manages.
func dpkgStates(ctx context.Context, m effect.Machine, names []string) (map[string]dpkgState, error) {
	args := append([]string{"-W", "-f", "${Package} ${Version} ${db:Status-Abbrev}\n"}, names...)
	out, err := m.Run(ctx, "dpkg-query", args...)
	// dpkg-query exits with 1 for a package it does not know, and prints the others.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, err
	}
	states := map[string]dpkgState{}
	for line := range strings.Lines(string(out)) {
		// A line of two fields has no version: dpkg knows of a package it never installed.
		f := strings.Fields(line)
		if len(f) != 3 {
			continue
		}
		// A package listed for two architectures keeps its installed line.
		if have, ok := states[f[0]]; !ok || !have.installed() {
			states[f[0]] = dpkgState{f[1], f[2]}
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
