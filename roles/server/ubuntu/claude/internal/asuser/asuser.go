// Package asuser reads a file under the user's home as the user, so that no link there
// reaches what only root could read.
package asuser

import (
	"context"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// readScript prints y and the file, or n where none exists. The mark tells a missing file from an
// empty one, and a failed cat fails the command.
const readScript = `if [ -e "$1" ]; then printf y; cat "$1"; else printf n; fi`

// Read reads a file under the user's home as the user, so that a link there reaches only what
// the user could read. The user's settings may link into a clone of dotfiles on purpose. A missing
// file returns fs.ErrNotExist.
func Read(ctx context.Context, m effect.Machine, u account.User, file string) ([]byte, error) {
	out, err := u.Output(ctx, m, "sh", "-c", readScript, "sh", file)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 || out[0] != 'y' {
		return nil, fs.ErrNotExist
	}
	return out[1:], nil
}
