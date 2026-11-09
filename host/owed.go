package host

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// owedDir holds a mark per unit whose restart or reload a watched change owes. A mark is written
// before the change and removed after the restart. So a run cut short, or one whose restart
// failed, leaves it, and the next run plans the restart again though the watched file is right.
const owedDir = "/var/lib/ws/owed"

// mark is a unit's file in owedDir.
type mark struct {
	m    effect.Machine
	name string
}

func (k mark) path() string { return path.Join(owedDir, k.name) }

// owed reports whether an earlier run left the mark.
func (k mark) owed() (bool, error) {
	_, err := k.m.RootAttrs(k.path())
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// settle removes the mark.
func (k mark) settle(ctx context.Context) error {
	_, err := k.m.RunRoot(ctx, "rm", "-f", "--", k.path())
	return err
}

// owe writes the mark. Group.Plan puts it before the changes that Watch lists.
type owe struct{ k mark }

func (c owe) String() string { return "note the owed restart of " + c.k.name }

func (owe) Kind() step.ChangeKind { return step.Write }

// Early moves the change before the rest of the group's.
func (owe) Early() bool { return true }

func (c owe) Apply(ctx context.Context) error {
	if _, err := c.k.m.RunRoot(ctx, "mkdir", "-p", "--", owedDir); err != nil {
		return err
	}
	return c.k.m.WriteRootFile(c.k.path(), nil, 0o600)
}
