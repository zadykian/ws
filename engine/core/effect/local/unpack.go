package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/zadykian/ws/engine/core/internal/archive"
)

// Unpack makes dir with mode 0755 where missing, and extracts the archive file into it through an
// os.Root, so that no entry or link leaves dir. Where ws runs as a user and may not write dir, or
// the nearest directory above it that exists, it extracts into a directory of ws's own. Then sudo
// copies that tree into dir, as root's.
func (m OSMachine) Unpack(ctx context.Context, file, dir string, strip int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !m.asRoot(dir) || !unwritable(existing(dir)) {
		return extractInto(file, dir, strip)
	}
	tmp, err := os.MkdirTemp("", ".ws-unpack-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }() // ws's own scratch tree
	if err := extractInto(file, tmp, strip); err != nil {
		return err
	}
	// -P copies a link as a link, and tmp/. copies tmp's entries, with GNU's cp and macOS's alike.
	for _, args := range [][]string{{"mkdir", "-p", "--", dir}, {"cp", "-RP", "--", tmp + "/.", dir}} {
		if _, err := m.RunRoot(ctx, args[0], args[1:]...); err != nil {
			return err
		}
	}
	return nil
}

// extractInto makes dir where missing, and extracts the archive file into it.
func extractInto(file, dir string, strip int) error {
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

// existing returns dir, or the nearest directory above it that exists.
func existing(dir string) string {
	for {
		if _, err := os.Lstat(dir); err == nil || filepath.Dir(dir) == dir {
			return dir
		}
		dir = filepath.Dir(dir)
	}
}
