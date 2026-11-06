package host

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// installOptions make apt-get wait for dpkg's lock, and keep the configuration file a machine
// already has where a package ships one, as Ansible's apt module did: a file ws wrote before the
// package came stays.
var installOptions = []string{"-o", "DPkg::Lock::Timeout=600",
	"-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold"}

// updateTries is how often apt-get update runs before the step fails.
const updateTries = 60

// Packages installs system packages, without recommends. Group it after the keyrings and
// .sources files that name their repositories. A new machine's apt timers hold apt's locks at
// first: apt-get update has no option to wait, so it tries again for 10 minutes, and apt-get
// install waits as long for dpkg's.
type Packages struct {
	M     effect.Machine
	Out   effect.Out
	Names []string
	// Recommends installs the recommended packages too, as apt does by default.
	Recommends bool
	// Sleep waits between the tries of apt-get update. Nil means time.Sleep.
	Sleep func(time.Duration)
}

// installChange installs the packages that are not installed.
type installChange struct {
	p       Packages
	missing []string
}

func (c installChange) String() string { return "install " + strings.Join(c.missing, " ") }

func (installChange) Kind() step.ChangeKind { return step.Install }

// Plan asks dpkg which of the packages are installed. A name of the form NAME=GLOB, such as
// nodejs=24.*, is installed only at a version that matches the glob.
func (p Packages) Plan(ctx context.Context) ([]Change, error) {
	var plain, pinned []string
	for _, n := range p.Names {
		if strings.Contains(n, "=") {
			pinned = append(pinned, n)
		} else {
			plain = append(plain, n)
		}
	}
	missing, err := p.missingPlain(ctx, plain)
	if err != nil {
		return nil, err
	}
	wrong, err := p.missingPinned(ctx, pinned)
	if err != nil {
		return nil, err
	}
	missing = append(missing, wrong...)
	if len(missing) == 0 {
		return nil, nil
	}
	return []Change{installChange{p, missing}}, nil
}

func (p Packages) missingPlain(ctx context.Context, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	args := append([]string{"-W", "-f", "${Package} ${db:Status-Abbrev}\n"}, names...)
	out, err := p.M.Run(ctx, "dpkg-query", args...)
	// dpkg-query exits with 1 for a package it does not know, and prints the others.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, err
	}
	return missingPackages(string(out), names), nil
}

// missingPinned returns those of the NAME=GLOB specs whose package is not installed at a version
// that the glob matches.
func (p Packages) missingPinned(ctx context.Context, specs []string) ([]string, error) {
	var missing []string
	for _, spec := range specs {
		name, glob, _ := strings.Cut(spec, "=")
		out, err := p.M.Run(ctx, "dpkg-query", "-W", "-f", "${Version} ${db:Status-Abbrev}\n", name)
		var ee *exec.ExitError
		if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
			return nil, err
		}
		if f := strings.Fields(string(out)); len(f) != 2 || f[1] != "ii" || !globMatch(glob, f[0]) {
			missing = append(missing, spec)
		}
	}
	return missing, nil
}

// globMatch reports whether version matches the glob, whose * stands for any run of characters.
func globMatch(glob, version string) bool {
	ok, err := path.Match(glob, version)
	return err == nil && ok
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

// Apply updates apt's lists, then installs the missing packages.
func (c installChange) Apply(ctx context.Context) error {
	p, missing := c.p, c.missing
	update := []string{"DEBIAN_FRONTEND=noninteractive", "apt-get", "update"}
	for try := 1; ; try++ {
		err := p.M.Pass(ctx, false, "env", update...)
		if err == nil {
			break
		}
		if try == updateTries {
			return fmt.Errorf("apt-get update failed %d times: %w", try, err)
		}
		p.Out.Say("apt-get update failed, trying again in 10 seconds")
		p.wait(10 * time.Second)
	}
	args := append(append([]string{"DEBIAN_FRONTEND=noninteractive", "apt-get"}, installOptions...), "install", "-y")
	if !p.Recommends {
		args = append(args, "--no-install-recommends")
	}
	return p.M.Pass(ctx, false, "env", append(args, missing...)...)
}

func (p Packages) wait(d time.Duration) {
	if p.Sleep != nil {
		p.Sleep(d)
		return
	}
	time.Sleep(d)
}

// Report shows the version of each package. A pin wants its glob.
func (p Packages) Report(ctx context.Context) ([]step.Item, error) {
	names := make([]string, len(p.Names))
	items := make([]step.Item, len(p.Names))
	for i, spec := range p.Names {
		name, glob, _ := strings.Cut(spec, "=")
		names[i] = name
		items[i] = step.Item{Kind: step.PackageItem, Name: name, Want: cmp.Or(glob, "installed")}
	}
	states, err := dpkgStates(ctx, p.M, names)
	if err != nil {
		return nil, err
	}
	for i, name := range names {
		items[i].Found = states[name].text()
	}
	return items, nil
}
