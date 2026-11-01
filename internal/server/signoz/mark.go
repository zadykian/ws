package signoz

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// mark is the file that keeps a recreation of the containers due until an up succeeds: after a
// failed up, the old containers may still run the old files. It comes before the changes it
// watches in the group, so that it is set before anything can fail.
type mark struct {
	M     effect.Machine
	Path  string
	Watch []host.Resource
}

type setMark struct{ mark }

func (setMark) String() string { return "mark SigNoz's containers for recreation" }

func (c setMark) Apply(context.Context) error { return writeMark(c.M, c.Path) }

func writeMark(m effect.Machine, path string) error { return m.WriteRootFile(path, nil, 0o600) }

// Plan sets the mark where it is not set and a watched resource has a change to make.
func (k mark) Plan(ctx context.Context) ([]host.Change, error) {
	if _, err := k.M.RootAttrs(k.Path); err == nil {
		return nil, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, r := range k.Watch {
		changes, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		if len(changes) > 0 {
			return []host.Change{setMark{k}}, nil
		}
	}
	return nil, nil
}

// markPath is the file that says the project's containers are to be recreated.
func markPath(dir string) string { return path.Join(dir, "recreate") }
