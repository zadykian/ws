package host

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// Package is a .deb package that the role pins, a name and a version, with the file of that
// version as Source. It fetches and installs the file where dpkg has another version, so that a
// newer file upgrades and an older one downgrades.
type Package struct {
	M      effect.Machine
	Name   string
	Source PinnedFile // its Path and Mode are not used
}

type installPackage struct {
	p    Package
	path string
}

func (c installPackage) String() string { return "install " + c.p.Name + " " + c.p.Source.Version }

func (installPackage) Kind() step.ChangeKind { return step.Install }

func (c installPackage) Apply(ctx context.Context) error {
	args := append([]string{"DEBIAN_FRONTEND=noninteractive", "apt-get"}, installOptions...)
	return c.p.M.Pass(ctx, false, "env", append(args, "install", "-y", "--allow-downgrades", c.path)...)
}

// Plan plans the directory, the fetch and the install where the pinned version is not installed.
func (p Package) Plan(ctx context.Context) ([]Change, error) {
	file, err := p.Source.cached(ctx)
	if err != nil {
		return nil, fmt.Errorf("package %s: %w", p.Name, err)
	}
	states, err := dpkgStates(ctx, p.M, []string{p.Name})
	if err != nil {
		return nil, err
	}
	if have := states[p.Name]; have.state == 'i' && have.version == p.Source.Version {
		return nil, nil
	}
	var changes []Change
	for _, r := range []Resource{Dir{M: p.M, Path: CacheDir, Mode: 0o755}, file} {
		more, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, more...)
	}
	return append(changes, installPackage{p, file.Path}), nil
}
