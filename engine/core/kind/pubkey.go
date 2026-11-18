package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// sshPub is the SSH public key at p, below user's home, as TYPE BASE64 with its comment dropped,
// as an allowed signers file takes it. The user owns the home, so the read refuses a link at the
// file or on the way. A missing file, as on a dry run before a unit makes the key, gives "".
func sshPub(ctx context.Context, e Env, user, p string) (string, error) {
	home, err := e.System.home(ctx, e.M, user)
	if err != nil {
		return "", err
	}
	rel, below := strings.CutPrefix(path.Clean(p), strings.TrimSuffix(home, "/")+"/")
	if !below {
		return "", fmt.Errorf("ssh_pub: %s is not below %s, the home of %s", p, home, user)
	}
	text, err := account.User{Name: user, Home: home}.ReadHome(e.M, rel)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	key, err := check.ParsePub(string(text))
	if err != nil {
		return "", fmt.Errorf("%s %w", p, err)
	}
	return key.String(), nil
}
