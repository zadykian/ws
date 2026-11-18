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
// logs in with it. The step generates it on the machine, and only root reads it. The containers
// read root.env at start: it sits in Dir, whose files the compose unit sums, so a new password
// recreates them.
func Password(m effect.Machine) resource.Group {
	return resource.Group{Title: "signoz password", Parts: []resource.Resource{password{M: m}}}
}

type password struct{ M effect.Machine }

func (password) path() string { return path.Join(Dir, "root.env") }

type generate struct{ password }

func (generate) String() string { return "generate the root user's password" }

func (c generate) Apply(context.Context) error {
	pw, err := randomPassword()
	if err != nil {
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
