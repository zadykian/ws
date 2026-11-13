package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

// programAndCopy plans ~/.local/bin/ws and root's copy of it. Every directory above the copy must
// be root's and writable by root alone, as one that another user could write would let that user
// replace the copy. /usr/local and /usr/local/libexec are made where missing. A rebuilt ws reaches
// the copy at the next run.
func (t *Tunnel) programAndCopy(ctx context.Context, self []byte) ([]step.Change, error) {
	cs, err := t.program(self)
	if err != nil {
		return nil, err
	}
	own := len(cs)
	for _, dir := range []string{"/usr/local", "/usr/local/libexec"} {
		if cs, err = change.Plan(ctx, cs, t.said(file.Dir{M: t.M, Path: dir, Mode: 0o755})); err != nil {
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
			return nil, fmt.Errorf("%s, above root's copy, is not root's alone: owner, group and mode %d:%d %o",
				dir, a.UID, a.GID, a.Mode.Perm())
		}
	}
	if cs, err = change.Plan(ctx, cs, t.said(file.Dir{M: t.M, Path: tunnelLibexec, Mode: 0o755})); err != nil {
		return nil, err
	}
	before := len(cs)
	// Secret: no diff shows a program.
	cs, err = change.Plan(ctx, cs, t.said(file.File{M: t.M, Path: tunnelLibexec + "/ws", Content: self, Mode: 0o755, Secret: true}))
	t.copied = len(cs) > before
	// Until Apply has made those changes, root's copy is not one to run as root.
	t.untrusted = len(cs) > own
	return cs, err
}
