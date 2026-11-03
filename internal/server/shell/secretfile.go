package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// secretEnv is ~/.secrets/env, for ~/.bashrc and the otelcol role, from the secrets file's
// [env] table. Without a secrets file, as in CI, ws leaves ~/.secrets alone, even a file it wrote,
// so that a run without it changes nothing. With one and an empty [env], ws removes the file it
// wrote.
type secretEnv struct {
	M   effect.Machine
	A   account
	Env map[string]string
	// Whole is the secrets file's say that [env] is all: with none, the file goes.
	Whole bool
}

// Plan checks the names, and that ws wrote the file there is. A file written by hand is refused
// rather than replaced: a backup would copy its secrets, and the new file would drop what the
// secrets file lacks, and so would a removal. A dry run stops there too. The file is read for its
// first line alone.
func (s secretEnv) Plan(ctx context.Context) ([]host.Change, error) {
	if len(s.Env) == 0 && !s.Whole {
		return nil, nil
	}
	if err := checkEnvNames(s.Env); err != nil {
		return nil, err
	}
	path := s.A.Home + "/.secrets/env"
	old, err := s.M.ReadRootFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && !bytes.HasPrefix(old, []byte(envMark)) {
		return nil, fmt.Errorf("%s lacks the line '%s' at its top, so it was written by hand: move its values into the secrets file's [env] table, remove the file, then run ws again", path, envMark)
	}
	if len(s.Env) == 0 {
		return planAll(ctx, host.Absent{M: s.M, Path: path})
	}
	return planAll(ctx,
		host.Dir{M: s.M, Path: s.A.Home + "/.secrets", Mode: 0o700, UID: s.A.UID, GID: s.A.GID},
		host.File{M: s.M, Path: path, Content: renderEnv(s.Env), Mode: 0o600, UID: s.A.UID, GID: s.A.GID, Secret: true},
	)
}
