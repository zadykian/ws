package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
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

// varsChange makes the machine's values, empty, where there are none.
type varsChange struct {
	path    string
	makeDir bool // the directory of the file is missing
}

func (c varsChange) String() string { return "write " + c.path + " with no values" }

// unpackChange unpacks the playbook into dir, which holds another content or none.
type unpackChange struct {
	dir, digest string
	makeLib     bool // the directory of dir is missing
}

func (c unpackChange) String() string { return "unpack the playbook into " + c.dir }

// pruneChange removes the entries of Lib that are not the version's.
type pruneChange struct{ paths []string }

func (c pruneChange) String() string { return "remove " + strings.Join(c.paths, ", ") }

// state is what Plan reads of the machine.
type state struct {
	varsMissing, varsDirMissing, libMissing bool
	unpacked                                string // the digest of Lib/VERSION, empty where none
	entries                                 []string
}

// changes decides what to do, from the digest of the embedded playbook and the state.
func changes(lib, vars, version, digest string, s state) []step.Change {
	var out []step.Change
	if s.varsMissing {
		out = append(out, varsChange{path: vars, makeDir: s.varsDirMissing})
	}
	dir := path.Join(lib, version)
	if s.unpacked != digest {
		out = append(out, unpackChange{dir: dir, digest: digest, makeLib: s.libMissing})
	}
	var others []string
	for _, e := range s.entries {
		if e != version {
			others = append(others, path.Join(lib, e))
		}
	}
	if len(others) > 0 {
		out = append(out, pruneChange{paths: others})
	}
	return out
}

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

// Plan reads the machine and makes the changes.
func (p Playbook) Plan(_ context.Context) (step.Plan, error) {
	if err := checkVersion(p.Version); err != nil {
		return step.Plan{}, err
	}
	if err := checkCollections(p.FS); err != nil {
		return step.Plan{}, err
	}
	digest, err := Digest(p.FS)
	if err != nil {
		return step.Plan{}, err
	}
	var s state
	if _, err := p.M.Stat(p.Vars); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return step.Plan{}, err
		}
		s.varsMissing = true
		if _, err := p.M.Stat(path.Dir(p.Vars)); errors.Is(err, fs.ErrNotExist) {
			s.varsDirMissing = true
		} else if err != nil {
			return step.Plan{}, err
		}
	}
	switch data, err := p.M.ReadFile(path.Join(p.Lib, p.Version, ".unpacked")); {
	case err == nil:
		s.unpacked = strings.TrimSpace(string(data))
	case !errors.Is(err, fs.ErrNotExist):
		return step.Plan{}, err
	}
	switch names, err := p.M.ReadDir(p.Lib); {
	case err == nil:
		s.entries = names
	case errors.Is(err, fs.ErrNotExist):
		s.libMissing = true
	default:
		return step.Plan{}, err
	}
	return step.Plan{Changes: changes(p.Lib, p.Vars, p.Version, digest, s)}, nil
}

// Apply makes the changes in the order of the plan: the values, the unpacking, the pruning.
func (p Playbook) Apply(ctx context.Context, pl step.Plan) error {
	for _, c := range pl.Changes {
		var err error
		switch c := c.(type) {
		case varsChange:
			err = p.writeVars(c)
		case unpackChange:
			err = p.unpack(ctx, c)
		case pruneChange:
			for _, e := range c.paths {
				if err = p.M.Pass(ctx, false, "rm", "-rf", e); err != nil {
					break
				}
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p Playbook) writeVars(c varsChange) error {
	if c.makeDir {
		if err := p.M.Mkdir(path.Dir(c.path), 0o755); err != nil {
			return err
		}
	}
	return p.M.WriteFile(c.path, []byte("---\n"), 0o600)
}
