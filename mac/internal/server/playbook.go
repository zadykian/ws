package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// versionName is what a version may hold, as it becomes a directory name under Lib.
var versionName = regexp.MustCompile(`^[A-Za-z0-9._+-]+$`)

// Playbook unpacks the playbook that this ws embeds into Lib/VERSION, with the collections
// installed and the machine's values linked in, and removes the other versions.
type Playbook struct {
	// FS is the embedded playbook: site.yml, requirements.yml and the collections' tarballs.
	FS fs.FS
	// Version is what ws version prints.
	Version string
	// Lib holds the unpacked playbooks, /var/lib/ws.
	Lib string
	// Vars is the machine's values, /etc/ws/host_vars.yml.
	Vars string
	M    effect.Machine
	Out  effect.Out
}

// Name returns the step's name.
func (Playbook) Name() string { return playbookStage }

// WaitsOn returns the packages, as ansible-galaxy comes with them.
func (Playbook) WaitsOn() string { return packagesStage }

// checkVersion refuses a version that is no directory name.
func checkVersion(v string) error {
	if !versionName.MatchString(v) || v == "." || v == ".." {
		return fmt.Errorf("the version %q is no name for a directory", v)
	}
	return nil
}

// checkCollections fails where the tarballs that ws embeds are not those requirements.yml pins.
func checkCollections(fsys fs.FS) error {
	data, err := fs.ReadFile(fsys, "requirements.yml")
	if err != nil {
		return err
	}
	pins, err := Pins(data)
	if err != nil {
		return err
	}
	tarballs, err := fs.Glob(fsys, "collections/*.tar.gz")
	if err != nil {
		return err
	}
	for i, t := range tarballs {
		tarballs[i] = path.Base(t)
	}
	if len(tarballs) == 0 && len(pins) > 0 {
		return errors.New("this ws holds no collections: build it with tools/build")
	}
	if !slices.Equal(pins, tarballs) {
		return errors.New("this ws's collections differ from requirements.yml's pins: build it again with tools/build")
	}
	return nil
}

// Plan checks the version and the collections.
func (p Playbook) Plan(_ context.Context) (step.Plan, error) {
	if err := checkVersion(p.Version); err != nil {
		return step.Plan{}, err
	}
	if err := checkCollections(p.FS); err != nil {
		return step.Plan{}, err
	}
	return step.Plan{}, nil
}
