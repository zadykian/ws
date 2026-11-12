package host

import (
	"io"
	"os"

	"github.com/zadykian/ws/internal/archive"
)

// Unpack extracts a tar archive, gzip-compressed or plain, into root, less the first strip
// components of each name. It writes through root alone, so no entry leaves it.
func Unpack(r io.Reader, root *os.Root, strip int) error {
	return archive.Extract(r, root, strip)
}

// UnpackFile makes dir with mode 0755, and unpacks the archive file in it as Unpack does.
func UnpackFile(file, dir string, strip int) error {
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
	defer func() { _ = root.Close() }() // the tree is written through Unpack, which reports its errors
	return Unpack(f, root, strip)
}
