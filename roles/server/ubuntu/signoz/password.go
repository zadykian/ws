package signoz

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// Password is the step that writes root.env once, with the password of SigNoz's root user. No one
// logs in with it. The step generates it on the machine, and only root reads it. A new password
// marks the containers for recreation, as they read root.env at start.
func Password(m effect.Machine, dir string) resource.Group {
	return resource.Group{
		Title: "signoz password", After: "signoz files",
		Parts: []resource.Resource{password{M: m, Dir: dir}},
	}
}

type password struct {
	M   effect.Machine
	Dir string
}

func (p password) path() string { return path.Join(p.Dir, "root.env") }

type generate struct{ password }

func (generate) String() string { return "generate the root user's password" }

func (c generate) Apply(context.Context) error {
	pw, err := randomPassword()
	if err != nil {
		return err
	}
	// The mark first, as a run cut short after the file would leave no mark for the new password.
	if err := writeMark(c.M, markPath(c.Dir)); err != nil {
		return err
	}
	return c.M.WriteRootFile(c.path(), []byte("SIGNOZ_USER_ROOT_PASSWORD="+pw+passwordSuffix+"\n"), 0o600)
}

// Plan generates the password where root.env is missing, and keeps the content of one that is
// there, whatever it holds, but its owner and mode.
func (p password) Plan(ctx context.Context) ([]resource.Change, error) {
	have, err := p.M.ReadRootFile(p.path())
	if errors.Is(err, fs.ErrNotExist) {
		return []resource.Change{generate{p}}, nil
	}
	if err != nil {
		return nil, err
	}
	return file.File{M: p.M, Path: p.path(), Content: have, Mode: 0o600, Secret: true}.Plan(ctx)
}
