// Package fetch downloads what ws pins: a file, an archive unpacked into a tree, and the machine's
// architecture for the release asset to take.
package fetch

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/internal/paths"
	"github.com/zadykian/ws/engine/core/step"
)

// Download is a file fetched from a URL. Its SHA-256, in hex, is pinned: the file is renamed into
// place only where the body matches.
type Download struct {
	M      effect.Machine
	URL    string
	Path   string
	SHA256 string
	Mode   fs.FileMode
	// UID and GID are the owner, root's by default.
	UID, GID int
}

func (d Download) attrs() effect.Attrs {
	return effect.Attrs{UID: d.UID, GID: d.GID, Mode: d.Mode.Perm()}
}

type fetch struct{ d Download }

func (c fetch) String() string { return fmt.Sprintf("download %s to %s", c.d.URL, c.d.Path) }

func (fetch) Kind() step.ChangeKind { return step.Write }

// downloadTries and retryWait are the fetch's: five tries, ten seconds apart, as a server can fail
// for a moment. A test shortens the wait.
const downloadTries = 5

var retryWait = 10 * time.Second

// Apply fetches the file again after a failure, except a wrong SHA-256, which a new try won't
// mend. The error is the last try's, with the earlier ones' kept.
func (c fetch) Apply(ctx context.Context) error {
	var errs []error
	for try := 1; try <= downloadTries; try++ {
		err := c.d.M.Download(ctx, c.d.URL, c.d.Path, c.d.SHA256, c.d.attrs())
		if err == nil {
			return nil
		}
		if errors.Is(err, effect.ErrSHA256) {
			return err
		}
		errs = append(errs, err)
		if try == downloadTries {
			break
		}
		select {
		case <-ctx.Done():
			return errors.Join(append(errs, ctx.Err())...)
		case <-time.After(retryWait):
		}
	}
	return errors.Join(errs...)
}

type setAttrs struct{ d Download }

func (c setAttrs) String() string {
	return fmt.Sprintf("set owner %d:%d and mode %o of %s", c.d.UID, c.d.GID, c.d.Mode.Perm(), c.d.Path)
}

func (setAttrs) Kind() step.ChangeKind { return step.Write }

func (c setAttrs) Apply(context.Context) error {
	if err := c.d.M.Chown(c.d.Path, c.d.UID, c.d.GID); err != nil {
		return err
	}
	return c.d.M.Chmod(c.d.Path, c.d.Mode)
}

// Plan hashes the file, and plans a download where the file is missing or its sum differs.
func (d Download) Plan(context.Context) ([]resource.Change, error) {
	body, err := d.M.ReadRootFile(d.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return []resource.Change{fetch{d}}, nil
	}
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(body)) != d.SHA256 {
		return []resource.Change{fetch{d}}, nil
	}
	got, err := d.M.RootAttrs(d.Path)
	if err != nil {
		return nil, err
	}
	if got != d.attrs() {
		return []resource.Change{setAttrs{d}}, nil
	}
	return nil, nil
}

// Report compares the file as Plan does, and one that differs stands against ws. Its content is a
// release's, so no diff shows it.
func (d Download) Report(context.Context) ([]step.Item, error) {
	item := step.Item{Kind: step.FileItem, Name: d.Path, Want: paths.AttrsText(d.attrs()), Found: paths.Missing}
	body, got, ok, err := paths.ReadRoot(d.M, d.Path)
	if !ok {
		return []step.Item{item}, err
	}
	item.Found = paths.AttrsText(got)
	switch {
	case fmt.Sprintf("%x", sha256.Sum256(body)) != d.SHA256:
		item.Found += ", other content"
		item.Conflict = "differs from the download ws pins: a change by hand, or another version of ws"
	case got != d.attrs():
		item.Conflict = paths.AttrsChanged
	}
	return []step.Item{item}, nil
}
