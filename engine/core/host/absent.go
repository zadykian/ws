package host

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host/internal/paths"
	"github.com/zadykian/ws/engine/core/step"
)

// Absent is a path that does not exist: a file, a link, or a directory with all it holds. Name only
// a path that ws owns.
type Absent struct {
	M    effect.Machine
	Path string
	// home, set by Account.Absent alone, is the home that the path lies below.
	home string
}

type removePath struct{ a Absent }

func (c removePath) String() string { return "remove " + c.a.Path }

func (removePath) Kind() step.ChangeKind { return step.Removal }

// Apply looks for a link on the way again, as a home file's does: root's rm resolves each directory
// on the way.
func (c removePath) Apply(ctx context.Context) error {
	if err := c.a.guard(); err != nil {
		return err
	}
	_, err := c.a.M.RunRoot(ctx, "rm", "-rf", "--", c.a.Path)
	return err
}

// guard refuses a link on the way to a path under the home. Another Absent is not checked.
func (a Absent) guard() error {
	if a.home == "" {
		return nil
	}
	return paths.GuardHome(a.M, a.home, a.Path)
}

// Plan plans the removal of a path that exists.
func (a Absent) Plan(context.Context) ([]Change, error) {
	if err := a.guard(); err != nil {
		return nil, err
	}
	_, err := a.M.RootAttrs(a.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []Change{removePath{a}}, nil
}

// Report says whether the path exists.
func (a Absent) Report(context.Context) ([]step.Item, error) {
	if err := a.guard(); err != nil {
		return nil, err
	}
	_, err := a.M.RootAttrs(a.Path)
	return reportPath(step.Item{Kind: step.FileItem, Name: a.Path, Want: paths.Missing, Found: paths.Missing}, paths.Present, "", err)
}
