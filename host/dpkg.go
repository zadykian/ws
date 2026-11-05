package host

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/effect"
)

// dpkgState is what dpkg-query says of a package. Its state is the status's second letter: n for
// nothing on the machine, c for settings alone. A package dpkg doesn't know has the zero state.
type dpkgState struct {
	version string
	state   byte
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
		if f := strings.Fields(line); len(f) == 3 && len(f[2]) >= 2 {
			states[f[0]] = dpkgState{f[1], f[2][1]}
		}
	}
	return states, nil
}

func (s dpkgState) installed() bool { return s.state != 0 && s.state != 'n' && s.state != 'c' }

func (s dpkgState) text() string {
	switch {
	case s.installed():
		return s.version
	case s.state == 'c':
		return "settings left"
	}
	return missing
}
