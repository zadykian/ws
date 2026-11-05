package host

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// Purged removes packages with their settings, and leaves those that aren't there.
type Purged struct {
	M     effect.Machine
	Names []string
}

type purge struct {
	m     effect.Machine
	names []string
}

func (c purge) String() string { return "purge " + strings.Join(c.names, " ") }

func (purge) Kind() step.ChangeKind { return step.Removal }

func (c purge) Apply(ctx context.Context) error {
	args := slices.Concat([]string{"DEBIAN_FRONTEND=noninteractive", "apt-get"}, installOptions, []string{"purge", "-y"}, c.names)
	return c.m.Pass(ctx, false, "env", args...)
}

// Plan reads dpkg's status. Its second letter is n for a package with nothing on the machine, and
// c, for one that left its settings, still takes a purge.
func (p Purged) Plan(ctx context.Context) ([]Change, error) {
	out, err := p.M.Run(ctx, "dpkg-query", append([]string{"-W", "-f", "${Package} ${db:Status-Abbrev}\n"}, p.Names...)...)
	// dpkg-query exits with 1 for a package it does not know, and prints the others.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, err
	}
	var present []string
	for line := range strings.Lines(string(out)) {
		if f := strings.Fields(line); len(f) == 2 && len(f[1]) >= 2 && f[1][1] != 'n' {
			present = append(present, f[0])
		}
	}
	if len(present) == 0 {
		return nil, nil
	}
	return []Change{purge{p.M, present}}, nil
}
