package host

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/zadykian/ws/effect"
)

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

// Plan asks dpkg which of the packages are installed.
func (p Packages) Plan(ctx context.Context) ([]Change, error) {
	args := append([]string{"-W", "-f", "${Package} ${db:Status-Abbrev}\n"}, p.Names...)
	out, err := p.M.Run(ctx, "dpkg-query", args...)
	// dpkg-query exits with 1 for a package it does not know, and prints the others.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, err
	}
	missing := missingPackages(string(out), p.Names)
	if len(missing) == 0 {
		return nil, nil
	}
	return []Change{installChange{p, missing}}, nil
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
	args := []string{"DEBIAN_FRONTEND=noninteractive", "apt-get", "-o", "DPkg::Lock::Timeout=600", "install", "-y"}
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
