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
// [env] table. Without it, as in CI, ws leaves ~/.secrets alone, even a file it wrote, so that
// a run without the secrets file changes nothing.
type secretEnv struct {
	M   effect.Machine
	A   account
	Env map[string]string
}

// Plan checks the names, and that ws wrote the file there is. A file written by hand is refused
// rather than replaced: a backup would copy its secrets, and the new file would drop what the
// secrets file lacks. A dry run stops there too. The file is read for its first line alone.
func (s secretEnv) Plan(ctx context.Context) ([]host.Change, error) {
	if len(s.Env) == 0 {
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
	return planAll(ctx,
		host.Dir{M: s.M, Path: s.A.Home + "/.secrets", Mode: 0o700, UID: s.A.UID, GID: s.A.GID},
		host.File{M: s.M, Path: path, Content: renderEnv(s.Env), Mode: 0o600, UID: s.A.UID, GID: s.A.GID, Secret: true},
	)
}
