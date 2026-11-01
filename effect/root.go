package effect

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// rootWrite fills the new file under umask 077, so that it is never readable beyond root before
// its mode is set, then gives it its owner and mode and renames it over the target.
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
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".ws-new")
	mode := strconv.FormatUint(uint64(perm.Perm()), 8)
	cmd := m.command(context.Background(),
		append(m.sudo(), "sh", "-c", rootWrite, "sh", tmp, owner, mode, path))
	cmd.Stdin = bytes.NewReader(data)
	if _, err := cmd.Output(); err != nil {
		_, _ = m.RunRoot(context.Background(), "rm", "-f", tmp)
		return commandError(cmd, err)
	}
	return nil
}

// ReadRootFile reads the file through sudo's cat.
func (m OSMachine) ReadRootFile(path string) ([]byte, error) {
	if _, err := m.RootAttrs(path); err != nil {
		return nil, err
	}
	return m.RunRoot(context.Background(), "cat", path)
}

// RootAttrs reads the owner and mode through sudo's stat.
func (m OSMachine) RootAttrs(path string) (Attrs, error) {
	out, err := m.RunRoot(context.Background(), "sh", "-c", rootStat, "sh", path)
	if err != nil {
		return Attrs{}, err
	}
	return parseAttrs(path, string(out))
}

func parseAttrs(path, out string) (Attrs, error) {
	out = strings.TrimSpace(out)
	if out == "missing" {
		return Attrs{}, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
	}
	var uid, gid int
	owner, bits, ok := strings.Cut(out, " ")
	if _, err := fmt.Sscanf(owner, "%d:%d", &uid, &gid); err != nil || !ok {
		return Attrs{}, fmt.Errorf("stat %s: unexpected output %q", path, out)
	}
	n, err := strconv.ParseUint(bits, 8, 32)
	if err != nil {
		return Attrs{}, fmt.Errorf("stat %s: unexpected output %q", path, out)
	}
	return Attrs{UID: uid, GID: gid, Mode: fs.FileMode(n).Perm()}, nil
}
