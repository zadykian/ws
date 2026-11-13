package apt

import (
	"context"
	"fmt"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/engine/core/resource/file"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Package is a .deb package that the role pins, a name and a version, with the file of that
// version as Source. It fetches and installs the file where dpkg has another version, so that a
// newer file upgrades and an older one downgrades. The install removes the file.
type Package struct {
	M      effect.Machine
	Name   string
	Source fetch.PinnedFile // its Path and Mode are not used
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
func (p Package) Plan(ctx context.Context) ([]resource.Change, error) {
	archive, err := p.Source.Cached(ctx)
	if err != nil {
		return nil, fmt.Errorf("package %s: %w", p.Name, err)
	}
	states, err := PackageStates(ctx, p.M, p.Name)
	if err != nil {
		return nil, err
	}
	if have := states[p.Name]; have.Installed() && have.Version == p.Source.Version {
		return file.Absent{M: p.M, Path: archive.Path}.Plan(ctx)
	}
	var changes []resource.Change
	for _, r := range []resource.Resource{file.Dir{M: p.M, Path: fetch.CacheDir, Mode: 0o755}, archive} {
		more, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, more...)
	}
	return append(changes, installPackage{p, archive.Path}), nil
}
