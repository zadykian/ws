package host

import (
	"context"
	"errors"
	"io/fs"
	"path"
)

// owedDir holds a mark per unit whose restart or reload a watched change owes. A mark is written
// before the change and removed after the restart, so that a run cut short, or one whose restart
// failed, leaves it, and the next run plans the restart again though the watched file is right.
const owedDir = "/var/lib/ws/owed"

func (u Unit) mark() string { return path.Join(owedDir, u.Name) }

// owed reports whether an earlier run left the unit's mark.
func (u Unit) owed() (bool, error) {
	_, err := u.M.RootAttrs(u.mark())
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// settle removes the unit's mark.
func (u Unit) settle(ctx context.Context) error {
	_, err := u.M.RunRoot(ctx, "rm", "-f", "--", u.mark())
	return err
}

// owe writes the unit's mark. Group.Plan puts it before the changes that Watch lists.
type owe struct{ u Unit }

func (c owe) String() string { return "note the owed restart of " + c.u.Name }

// Early moves the change before the rest of the group's.
func (owe) Early() bool { return true }

func (c owe) Apply(ctx context.Context) error {
	if _, err := c.u.M.RunRoot(ctx, "mkdir", "-p", "--", owedDir); err != nil {
		return err
	}
	return c.u.M.WriteRootFile(c.u.mark(), nil, 0o600)
}
