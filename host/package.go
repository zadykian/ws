package host

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// Package is a .deb package that the role pins, a name and a version, with the file of that
// version as Source. It fetches and installs the file where dpkg has another version, so that a
// newer file upgrades and an older one downgrades. The install removes the file.
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
	if err := c.check(ctx); err != nil {
		return err
	}
	args := append([]string{"DEBIAN_FRONTEND=noninteractive", "apt-get"}, installOptions...)
	if err := c.p.M.Pass(ctx, false, "env", append(args, "install", "-y", "--allow-downgrades", c.path)...); err != nil {
		return err
	}
	_, err := c.p.M.RunRoot(ctx, "rm", "-f", "--", c.path)
	return err
}

// check reads the file's own name and version, so that a file of another package or version,
// whatever its URL says, never reaches dpkg.
func (c installPackage) check(ctx context.Context) error {
	out, err := c.p.M.Run(ctx, "dpkg-deb", "-f", c.path, "Package", "Version")
	if err != nil {
		return fmt.Errorf("package %s: %w", c.p.Name, err)
	}
	var name, version string
	for line := range strings.Lines(string(out)) {
		if v, ok := strings.CutPrefix(line, "Package: "); ok {
			name = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "Version: "); ok {
			version = strings.TrimSpace(v)
		}
	}
	if name != c.p.Name || version != c.p.Source.Version {
		return fmt.Errorf("package %s: the file is %s %s, want %s %s", c.p.Name, name, version, c.p.Name, c.p.Source.Version)
	}
	return nil
}

// Plan plans the directory, the fetch and the install where the pinned version is not installed.
// With the pinned version installed, Plan plans the removal of a file that a cut run left.
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
		return Absent{M: p.M, Path: file.Path}.Plan(ctx)
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
