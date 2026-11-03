package jetbrains

import (
	"context"
	"fmt"
	"strings"
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
		_, rmErr := c.s.AsUser(ctx, "rm", "-rf", "--", c.s.Downloads)
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
		return c.download(ctx)
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
		_, err := c.s.AsUser(ctx, "rm", "-rf", "--", c.b.Dir, c.b.Dir+".part")
		return err
	})
	return done, err
}

// download fetches the archive as the user, and removes it where its SHA-256 is another. The user
// is the one who unpacks it, so the check holds for what the user's tar reads.
func (c install) download(ctx context.Context) error {
	if _, err := c.s.AsUser(ctx, "curl", "-fsSL", "-o", c.archive(), "--url", c.b.Link); err != nil {
		return err
	}
	out, err := c.s.AsUser(ctx, "sha256sum", "--", c.archive())
	if err != nil {
		return err
	}
	if got, _, _ := strings.Cut(string(out), " "); got != c.sum {
		_, _ = c.s.AsUser(ctx, "rm", "-f", "--", c.archive())
		return fmt.Errorf("download %s: SHA-256 is %s, want %s", c.b.Link, got, c.sum)
	}
	return nil
}
