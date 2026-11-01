package server

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// packages are what the playbook needs of the system. They install without recommends:
// ansible-core's bring Ubuntu's ansible package, whose collections would stand in for any
// requirements.yml lacks. python3-apt is named because Ansible's apt module installs it itself
// only in a real run, not a dry one. ca-certificates serves the playbook's downloads over HTTPS.
var packages = []string{"ansible-core", "ca-certificates", "python3-apt"}

// Packages installs the system packages that the playbook needs.
type Packages struct {
	M   effect.Machine
	Out effect.Out
}

// Name returns the step's name.
func (Packages) Name() string { return packagesStage }

// WaitsOn returns no step: the packages come first.
func (Packages) WaitsOn() string { return "" }

// installChange installs the packages that are not installed.
type installChange struct{ missing []string }

func (c installChange) String() string { return "install " + strings.Join(c.missing, " ") }

// Plan asks dpkg which of the packages are installed.
func (p Packages) Plan(ctx context.Context) (step.Plan, error) {
	args := append([]string{"-W", "-f", "${Package} ${db:Status-Abbrev}\n"}, packages...)
	out, err := p.M.Run(ctx, "dpkg-query", args...)
	// dpkg-query exits with 1 for a package it does not know, and prints the others.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return step.Plan{}, err
	}
	missing := missingPackages(string(out), packages)
	if len(missing) == 0 {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{installChange{missing}}}, nil
}

// missingPackages returns those of want that dpkg-query's output does not list as installed, "ii".
func missingPackages(out string, want []string) []string {
	installed := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[1] == "ii" {
			installed[f[0]] = true
		}
	}
	var missing []string
	for _, w := range want {
		if !installed[w] {
			missing = append(missing, w)
		}
	}
	return missing
}
