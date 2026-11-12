package host

import (
	"context"
	"io"
	"os"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/archive"
)

// Unpack extracts a tar archive, gzip-compressed or plain, into root, less the first strip
// components of each name. It writes through root alone, so no entry leaves it.
func Unpack(r io.Reader, root *os.Root, strip int) error {
	return archive.Extract(r, root, strip)
}

// UnpackFile makes dir with mode 0755, and unpacks the archive file in it as Unpack does.
func UnpackFile(file, dir string, strip int) error {
	return effect.OSMachine{}.Unpack(context.Background(), file, dir, strip)
}
