package local

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/zadykian/ws/engine/core/effect"
)

// rootWrite fills the new file under umask 077, so that no one but root reads it before its mode
// is set. Then it gives the file its owner and mode and renames it over the target.
const rootWrite = `umask 077 && cat >"$1" && chown "$2" "$1" && chmod "$3" "$1" && mv -f "$1" "$4"`

// rootStat prints "uid:gid mode" of a file, or "missing". GNU's stat comes first: BSD's -f prints a file system's data under GNU.
const rootStat = `[ -e "$1" ] || { echo missing; exit 0; }
stat -c '%u:%g %a' "$1" 2>/dev/null || stat -f '%u:%g %Lp' "$1"`

// WriteRootFile writes a new file beside path, through sudo, then renames it over path.
func (m OSMachine) WriteRootFile(path string, data []byte, perm fs.FileMode) error {
	owner := m.Owner
	if owner == "" {
		owner = "0:0"
	}
	return m.sudoWrite(path, data, owner, perm)
}

// ReadRootFile reads the file through sudo's cat, or as ws's user with UserReads.
func (m OSMachine) ReadRootFile(path string) ([]byte, error) {
	if _, err := m.RootAttrs(path); err != nil {
		return nil, err
	}
	if m.UserReads {
		data, err := os.ReadFile(path) //nolint:gosec // the path is a unit's own, not input
		return data, unreadable(path, err)
	}
	return m.RunRoot(context.Background(), "cat", path)
}

// RootAttrs reads the owner and mode through sudo's stat, or as ws's user with UserReads.
func (m OSMachine) RootAttrs(path string) (effect.Attrs, error) {
	if m.UserReads {
		return userAttrs(path)
	}
	out, err := m.RunRoot(context.Background(), "sh", "-c", rootStat, "sh", path)
	if err != nil {
		return effect.Attrs{}, err
	}
	return parseAttrs(path, string(out))
}

// userAttrs is rootStat's answer, read as ws's user. A path whose link leads nowhere is missing. A
// link's own owner and mode stand for it, as stat's do without -L.
func userAttrs(path string) (effect.Attrs, error) {
	if _, err := os.Stat(path); err != nil {
		return effect.Attrs{}, unreadable(path, err)
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return effect.Attrs{}, unreadable(path, err)
	}
	a := effect.Attrs{Mode: fi.Mode().Perm()}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		a.UID, a.GID = int(st.Uid), int(st.Gid)
	}
	return a, nil
}

// unreadable says why a read as ws's user failed where the user lacks the right, as no sudo
// follows it.
func unreadable(path string, err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("%s: root's file is not readable by ws's user, and a read asks no sudo: "+
			"keep root's paths readable by all: %w", path, err)
	}
	return err
}

func parseAttrs(path, out string) (effect.Attrs, error) {
	out = strings.TrimSpace(out)
	if out == "missing" {
		return effect.Attrs{}, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	var uid, gid int
	owner, bits, ok := strings.Cut(out, " ")
	if _, err := fmt.Sscanf(owner, "%d:%d", &uid, &gid); err != nil || !ok {
		return effect.Attrs{}, fmt.Errorf("stat %s: unexpected output %q", path, out)
	}
	n, err := strconv.ParseUint(bits, 8, 32)
	if err != nil {
		return effect.Attrs{}, fmt.Errorf("stat %s: unexpected output %q", path, out)
	}
	return effect.Attrs{UID: uid, GID: gid, Mode: fs.FileMode(n).Perm()}, nil
}
