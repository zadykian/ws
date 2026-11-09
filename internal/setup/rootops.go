package setup

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

// removeRoot plans the removal of root's file, if the file is there.
func removeRoot(m effect.Machine, file string) (step.Change, error) {
	have, err := m.ReadRootFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return RootRemove{file, have}, nil
}

// applyInOrder applies the changes of a plan in order, with launchctl's signal as a change.
func applyInOrder(ctx context.Context, m effect.Machine, out effect.Out, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		if l, ok := c.(Launchctl); ok {
			err = launchctl(ctx, m, out, time.Sleep, l)
		} else {
			err = Apply(ctx, m, out, c)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// plan appends the changes that r plans to cs.
func plan(ctx context.Context, cs []step.Change, r host.Resource) ([]step.Change, error) {
	changes, err := r.Plan(ctx)
	for _, c := range changes {
		cs = append(cs, c)
	}
	return cs, err
}

// removal is host's removal of a file that no Report names, as mac.conf no longer does. It says
// what the file's Report would.
type removal struct {
	host.Change
	path string
}

// Kind is removal.
func (removal) Kind() step.ChangeKind { return step.Removal }

// Item says that the file is to go.
func (r removal) Item() step.Item {
	return step.Item{Kind: step.FileItem, Name: r.path, Want: stateMissing, Found: statePresent}
}
