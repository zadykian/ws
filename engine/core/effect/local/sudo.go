package local

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// euid is the user ws runs as, and access checks what it may do to a path. Variables, so that a
// test runs as another user than its own.
var (
	euid   = os.Geteuid
	access = unix.Access
)

// asRoot is whether a change to path may go through sudo: ws runs as another user than root, and
// path lies outside the user's home. In the home, or with no home known, a change ws may not make
// fails: sudo would leave root's files where the user's go.
func (OSMachine) asRoot(path string) bool {
	home, err := os.UserHomeDir()
	return euid() != 0 && err == nil && !under(home, path)
}

// under is whether path lies below dir, or is dir, by their text. An empty dir holds nothing.
func under(dir, path string) bool {
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	return err == nil && filepath.IsLocal(rel)
}

// another is whether uid, where 0 or more, or the owner of path, is another user than ws's: a
// change to either needs root. A path that ws may not stat is another's. A link is not, as sudo
// would follow it: the change goes as ws's, and fails where ws may not make it.
func another(path string, uid int) bool {
	fi, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrPermission):
		return true
	case err == nil && fi.Mode()&fs.ModeSymlink != 0:
		return false
	case uid >= 0 && uid != euid():
		return true
	case err != nil:
		return false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) != euid()
}

// unwritable is whether ws may not make a file in dir. A directory that all may write, as /tmp, is
// root's but takes ws's own files.
func unwritable(dir string) bool {
	return errors.Is(access(dir, unix.W_OK), fs.ErrPermission)
}

// sudoWrite fills a new file beside path through sudo, gives it owner, as uid:gid, and perm, and
// renames it over path.
func (m OSMachine) sudoWrite(path string, data []byte, owner string, perm fs.FileMode) error {
	return m.sudoWriteFrom(path, bytes.NewReader(data), owner, perm)
}

// sudoWriteFrom is sudoWrite for content that a reader holds, which a download's file is.
func (m OSMachine) sudoWriteFrom(path string, data io.Reader, owner string, perm fs.FileMode) error {
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".ws-new")
	mode := strconv.FormatUint(uint64(perm.Perm()), 8)
	cmd := m.command(context.Background(),
		append(m.sudo(), "sh", "-c", rootWrite, "sh", tmp, owner, mode, path))
	cmd.Stdin = data
	if _, err := cmd.Output(); err != nil {
		_, _ = m.RunRoot(context.Background(), "rm", "-f", tmp)
		return commandError(cmd, err)
	}
	return nil
}

// sudoMkdir makes the directory through sudo, with exactly perm, as mkdir's -m ignores the umask. A
// path that exists fails as os.Mkdir's does, which a caller tells apart.
func (m OSMachine) sudoMkdir(path string, perm fs.FileMode) error {
	if _, err := os.Lstat(path); err == nil {
		return &fs.PathError{Op: "mkdir", Path: path, Err: fs.ErrExist}
	}
	_, err := m.RunRoot(context.Background(), "mkdir", "-m", strconv.FormatUint(uint64(perm.Perm()), 8), "--", path)
	return err
}
