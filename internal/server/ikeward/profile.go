package ikeward

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/secrets"
)

// runDir is where the profile's files go for one run: in /run, where only root may enter. They keep
// their names, which name the files that install writes.
const runDir = "/run/ikeward-profile"

// profile is the profile's files from the secrets, and the names they are written under.
type profile struct {
	m     effect.Machine
	name  string // the profile's path less the extension
	files secrets.Files
}

// present reports whether the secrets hold both files.
func (p profile) present() bool { return len(p.files.P12) > 0 && len(p.files.Sswan) > 0 }

// base is the path of the written files, less the extension.
func (p profile) base() string { return path.Join(runDir, path.Base(p.name)) }

// with writes the files, with no output of their names or bytes, runs f, and removes them whether f
// fails or not, even those a run killed before left in the directory.
func (p profile) with(ctx context.Context, f func() error) (err error) {
	defer func() {
		_, rmErr := p.m.RunRoot(ctx, "rm", "-rf", "--", runDir)
		err = errors.Join(err, rmErr)
	}()
	if err := p.m.Mkdir(runDir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	if err := p.m.Chmod(runDir, 0o700); err != nil {
		return err
	}
	for ext, data := range map[string][]byte{".p12": p.files.P12, ".sswan": p.files.Sswan} {
		if err := p.m.WriteRootFile(p.base()+ext, data, 0o600); err != nil {
			return err
		}
	}
	return f()
}
