package jetbrains

import (
	"context"

	"github.com/zadykian/ws/effect"
)

func (c install) String() string { return c.b.Name + " would be installed in " + c.b.Dir }

// Apply removes what an install cut short left, checks the room, downloads the archive and
// unpacks it. The downloads' directory goes with its archive, whether the install succeeds or not.
func (c install) Apply(ctx context.Context) (err error) {
	done, err := c.clear(ctx)
	if err != nil || done {
		return err
	}
	if err := c.room(); err != nil {
		return err
	}
	changes, err := Dirs{S: c.s, Path: c.s.Downloads}.Plan(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_, rmErr := c.s.M.Run(ctx, "rm", "-rf", "--", c.s.Downloads)
		if err == nil {
			err = rmErr
		}
	}()
	for _, ch := range changes {
		if err := ch.Apply(ctx); err != nil {
			return err
		}
	}
	err = c.s.retry(ctx, func() error {
		return c.s.M.Download(ctx, c.b.Link, c.archive(), c.sum, c.attrs())
	})
	if err != nil {
		return err
	}
	return c.s.Lock(c.s.Dist, func() error { return c.unpack(ctx) })
}

// clear removes the build's directory and DIR.part, which an install cut short left, the role's or
// Gateway's, before the room is measured, as their space is freed. It reports whether the build
// is in place, as another run may have installed it by the time this one holds the lock.
func (c install) clear(ctx context.Context) (done bool, err error) {
	err = c.s.Lock(c.s.Dist, func() error {
		if _, err := c.s.M.Stat(c.b.Dir + "/" + marker); err == nil {
			done = true
			return nil
		}
		_, err := c.s.M.Run(ctx, "rm", "-rf", "--", c.b.Dir, c.b.Dir+".part")
		return err
	})
	return done, err
}

func (c install) attrs() effect.Attrs { return effect.Attrs{UID: c.s.UID, GID: c.s.GID, Mode: 0o644} }
