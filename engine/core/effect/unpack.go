package effect

import (
	"context"
	"errors"
	"os"

	"github.com/zadykian/ws/engine/core/internal/archive"
)

// Unpack makes dir with mode 0755 where missing, and extracts the archive file into it through an
// os.Root, so that no entry or link leaves dir.
func (OSMachine) Unpack(ctx context.Context, file, dir string, strip int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f, err := os.Open(file) //nolint:gosec // the step names the archive
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()                // read only
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // a tree for all users to read
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	err = archive.Extract(f, root, strip)
	return errors.Join(err, root.Close())
}
