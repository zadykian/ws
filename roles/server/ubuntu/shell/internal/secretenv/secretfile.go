package secretenv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/server/ubuntu/internal/plan"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// Env is /etc/ws/env, for ~/.bashrc and the otelcol role, from the secrets file's [env] table.
// Root owns it, with the account's group, so that the account reads it and cannot change it.
// Without a secrets file, as in CI, ws leaves it alone, even a file it wrote, so that a run without
// it changes nothing. With one and an empty [env], ws removes the file it wrote.
type Env struct {
	M   effect.Machine
	A   account.Account
	Env map[string]string
	// Whole is the secrets file's say that [env] is all: with none, the file goes.
	Whole bool
}

// Plan checks the names, and that ws wrote the file there is. A file written by hand is refused
// rather than replaced. A backup would copy its secrets, and the new file would drop what the
// secrets file lacks, and so would a removal. A dry run stops there too. The file is read for its
// first line alone.
func (s Env) Plan(ctx context.Context) ([]resource.Change, error) {
	if len(s.Env) == 0 && !s.Whole {
		return nil, nil
	}
	if err := checkEnvNames(s.Env); err != nil {
		return nil, err
	}
	old, err := s.M.ReadRootFile(secrets.EnvPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && !bytes.HasPrefix(old, []byte(envMark)) {
		return nil, fmt.Errorf("%s lacks the line '%s' at its top, so it was written by hand: move its values into the secrets file's [env] table, remove the file, then run ws again", secrets.EnvPath, envMark)
	}
	if len(s.Env) == 0 {
		return plan.All(ctx, file.Absent{M: s.M, Path: secrets.EnvPath})
	}
	return plan.All(ctx, file.File{M: s.M, Path: secrets.EnvPath, Content: renderEnv(s.Env),
		Mode: 0o640, GID: s.A.GID, Secret: true})
}
