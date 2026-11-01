package setup

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/mac/internal/step"
)

// programAndCopy plans ~/.local/bin/ws and root's copy of it. Every directory above the copy must
// be root's and writable by root alone, as one that another user could write would let that user
// replace the copy. /usr/local and /usr/local/libexec are made where missing. A rebuilt ws reaches
// the copy at the next run.
func (t *Tunnel) programAndCopy(self []byte) ([]step.Change, error) {
	var cs []step.Change
	add := func(c step.Change, err error) error {
		if err == nil && c != nil {
			cs = append(cs, c)
		}
		return err
	}
	if err := add(t.program(self)); err != nil {
		return nil, err
	}
	own := len(cs)
	for _, dir := range []string{"/usr/local", "/usr/local/libexec"} {
		if err := add(rootDir(t.M, dir, 0o755)); err != nil {
			return nil, err
		}
	}
	for _, dir := range []string{"/", "/usr", "/usr/local", "/usr/local/libexec"} {
		a, err := t.M.RootAttrs(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if a.UID != 0 || a.Mode.Perm()&0o022 != 0 {
			return nil, fmt.Errorf("%s, above root's copy, is not root's alone: owner, group and mode %s", dir, attrsText(a))
		}
	}
	if err := add(rootDir(t.M, tunnelLibexec, 0o755)); err != nil {
		return nil, err
	}
	copied, err := rootFile(t.M, tunnelLibexec+"/ws", 0o755, self)
	if err != nil {
		return nil, err
	}
	if w, ok := copied.(RootWrite); ok {
		copied = RootCopy{From: t.Home + "/.local/bin/ws", To: w.Path, Data: self, Mode: 0o755}
	}
	t.copied = copied != nil
	err = add(copied, nil)
	// Until Apply has made those changes, root's copy is not one to run as root.
	t.untrusted = len(cs) > own
	return cs, err
}
