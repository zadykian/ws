package host

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// Absent is a path that does not exist: a file, a link, or a directory with all it holds. Name only
// a path that ws owns.
type Absent struct {
	M    effect.Machine
	Path string
}

type removePath struct{ a Absent }

func (c removePath) String() string { return "remove " + c.a.Path }

func (removePath) Kind() step.ChangeKind { return step.Removal }

func (c removePath) Apply(ctx context.Context) error {
	_, err := c.a.M.RunRoot(ctx, "rm", "-rf", "--", c.a.Path)
	return err
}

// Plan plans the removal of a path that exists.
func (a Absent) Plan(context.Context) ([]Change, error) {
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
	_, err := a.M.RootAttrs(a.Path)
	return reportPath(step.Item{Kind: step.FileItem, Name: a.Path, Want: missing, Found: missing}, present, "", err)
}
