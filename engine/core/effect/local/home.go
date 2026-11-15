package local

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/zadykian/ws/engine/core/effect"
)

// ReadHomeFile opens the home, then each directory below it and the file, in ws's own process. The
// open of a directory refuses a link and a FIFO. The open of the file refuses a link and does not
// wait on a FIFO. So the account cannot lead a root reader elsewhere or hold it, even by a swap
// after a check. The bytes and the owner and mode come from the file's one descriptor.
func (OSMachine) ReadHomeFile(home, path string) ([]byte, effect.Attrs, error) {
	rel, err := filepath.Rel(home, path)
	if err != nil || !filepath.IsLocal(rel) {
		return nil, effect.Attrs{}, fmt.Errorf("%s is not a path below the home %s", path, home)
	}
	names := strings.Split(filepath.Clean(rel), string(filepath.Separator))
	at, err := unix.Open(home, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, effect.Attrs{}, &fs.PathError{Op: "open", Path: path, Err: err}
	}
	for _, name := range names[:len(names)-1] {
		next, err := unix.Openat(at, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(at)
		if err != nil {
			return nil, effect.Attrs{}, &fs.PathError{Op: "open", Path: path, Err: err}
		}
		at = next
	}
	fd, err := unix.Openat(at, names[len(names)-1], unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	_ = unix.Close(at)
	if err != nil {
		return nil, effect.Attrs{}, &fs.PathError{Op: "open", Path: path, Err: err}
	}
	f := os.NewFile(uintptr(fd), path)
	defer func() { _ = f.Close() }()
	return readOpened(f, path)
}

// readOpened reads the open file if regular and small, with the owner and mode of the same
// descriptor.
func readOpened(f *os.File, path string) ([]byte, effect.Attrs, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, effect.Attrs{}, &fs.PathError{Op: "read", Path: path, Err: err}
	}
	if !fi.Mode().IsRegular() {
		return nil, effect.Attrs{}, &fs.PathError{Op: "read", Path: path, Err: fmt.Errorf("%w: %v", effect.ErrNotRegular, fi.Mode().Type())}
	}
	tooLarge := &fs.PathError{Op: "read", Path: path, Err: fmt.Errorf("%w: over %d bytes", effect.ErrTooLarge, effect.HomeFileMax)}
	if fi.Size() > effect.HomeFileMax {
		return nil, effect.Attrs{}, tooLarge
	}
	data, err := io.ReadAll(io.LimitReader(f, effect.HomeFileMax+1))
	if err != nil {
		return nil, effect.Attrs{}, &fs.PathError{Op: "read", Path: path, Err: err}
	}
	if len(data) > effect.HomeFileMax {
		return nil, effect.Attrs{}, tooLarge
	}
	a := effect.Attrs{Mode: fi.Mode().Perm()}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		a.UID, a.GID = int(st.Uid), int(st.Gid)
	}
	return data, a, nil
}
