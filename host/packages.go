package host

import (
	"cmp"
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// installOptions make apt-get wait for dpkg's lock. They keep the configuration file a machine
// already has where a package ships one, as Ansible's apt module did: a file ws wrote before the
// package came stays.
var installOptions = []string{"-o", "DPkg::Lock::Timeout=600",
	"-o", "Dpkg::Options::=--force-confdef", "-o", "Dpkg::Options::=--force-confold"}

// updateTries is how often apt-get update runs before the step fails.
const updateTries = 60

// Packages installs system packages, without recommends. Group it after the keyrings and
// .sources files that name their repositories. A new machine's apt timers hold apt's locks at
// first. apt-get update has no option to wait, so it tries again for 10 minutes, and apt-get
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

// Plan asks dpkg, in one run, which of the packages are installed. A name of the form NAME=GLOB,
// such as nodejs=24.*, is installed only at a version that matches the glob. The change lists
// the plain names first, then the pins.
func (p Packages) Plan(ctx context.Context) ([]Change, error) {
	var plain, pinned, patterns []string
	for _, n := range p.Names {
		if name, _, ok := strings.Cut(n, "="); ok {
			pinned = append(pinned, n)
			patterns = append(patterns, name)
		} else {
			plain = append(plain, n)
			patterns = append(patterns, n)
		}
	}
	states, err := dpkgStates(ctx, p.M, patterns...)
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, n := range plain {
		if !states[n].installed() {
			missing = append(missing, n)
		}
	}
	for _, spec := range pinned {
		name, glob, _ := strings.Cut(spec, "=")
		if have := states[name]; !have.installed() || !globMatch(glob, have.version) {
			missing = append(missing, spec)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	return []Change{installChange{p, missing}}, nil
}

// globMatch reports whether version matches the glob, whose * stands for any run of characters.
func globMatch(glob, version string) bool {
	ok, err := path.Match(glob, version)
	return err == nil && ok
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
	states, err := dpkgStates(ctx, p.M, names...)
	if err != nil {
		return nil, err
	}
	for i, name := range names {
		items[i].Found = states[name].text()
	}
	return items, nil
}
