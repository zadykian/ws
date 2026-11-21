package kind

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// drift finds what is in force other than what a service's files load, as after a run cut short
// before the restart, or a change by hand. Nothing else would restart the service then. inForce
// and loads are two commands, run as root, whose outputs match where the two agree. A load that
// fails, or nothing in force, counts as a difference: the restart shows which. A file not yet
// written has nothing to load, and its write is a change of its own.
type drift struct {
	m       effect.Machine
	inForce []string
	loads   []string
	files   []string
}

type restartDue struct{ inForce []string }

func (c restartDue) String() string {
	return "restart: what " + strings.Join(c.inForce, " ") + " prints is not what the files load"
}

// Apply does nothing: the service that watches the drift restarts.
func (restartDue) Apply(context.Context) error { return nil }

// ByRestart lets the restart go on while the drift stays, as the restart is what settles it.
func (restartDue) ByRestart() bool { return true }

// Plan plans a restart where the two outputs differ.
func (d drift) Plan(ctx context.Context) ([]resource.Change, error) {
	for _, p := range d.files {
		_, err := d.m.RootAttrs(p)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}
	due := []resource.Change{restartDue{d.inForce}}
	loaded, err := d.m.RunRoot(ctx, d.loads[0], d.loads[1:]...)
	if err != nil {
		return due, nil //nolint:nilerr // files that fail to load are a restart due
	}
	inForce, err := d.m.RunRoot(ctx, d.inForce[0], d.inForce[1:]...)
	if err != nil || !bytes.Equal(inForce, loaded) {
		return due, nil //nolint:nilerr // nothing in force is a restart due
	}
	return nil, nil
}
