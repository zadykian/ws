package host

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/effect"
)

// Deb is a package from a .deb file on disk, such as a download. It installs where the package is
// missing or its version differs from the file's, so that a newer file upgrades and an older one
// downgrades. Put the file's Download before it in the group.
type Deb struct {
	M    effect.Machine
	Path string // absolute
}

type installDeb struct {
	d    Deb
	name string
}

func (c installDeb) String() string { return "install " + c.name + " from " + c.d.Path }

func (c installDeb) Apply(ctx context.Context) error {
	args := append([]string{"DEBIAN_FRONTEND=noninteractive", "apt-get"}, installOptions...)
	return c.d.M.Pass(ctx, false, "env", append(args, "install", "-y", "--allow-downgrades", c.d.Path)...)
}

// Plan reads the file's package and version with dpkg-deb, and the installed version with dpkg.
func (d Deb) Plan(ctx context.Context) ([]Change, error) {
	if !filepath.IsAbs(d.Path) {
		return nil, fmt.Errorf("deb %s: the path is not absolute", d.Path)
	}
	out, err := d.M.Run(ctx, "dpkg-deb", "-f", d.Path, "Package", "Version")
	if err != nil {
		return nil, fmt.Errorf("deb %s: %w", d.Path, err)
	}
	var name, version string
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, "Package: "); ok {
			name = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "Version: "); ok {
			version = strings.TrimSpace(v)
		}
	}
	if name == "" || version == "" {
		return nil, fmt.Errorf("deb %s: no package or version in %q", d.Path, out)
	}
	have, err := d.M.Run(ctx, "dpkg-query", "-W", "-f", "${Version} ${db:Status-Abbrev}\n", name)
	// dpkg-query exits with 1 for a package it does not know.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return nil, err
	}
	if strings.TrimSpace(string(have)) == version+" ii" {
		return nil, nil
	}
	return []Change{installDeb{d, name}}, nil
}
