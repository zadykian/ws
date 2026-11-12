package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
)

// Apply downloads the install.sh of the latest release into a directory of its own, runs it, and
// replaces this process with the new ws, run with the same arguments. The new ws installs its
// version on ws, and runs ws server apply again.
func (u *MacUpdate) Apply(ctx context.Context, p step.Plan) error {
	tag := p.Changes[0].(macUpdate).tag
	dir, err := u.M.Run(ctx, "mktemp", "-d")
	if err != nil {
		return fmt.Errorf("making a directory for install.sh: %w", err)
	}
	tmp := strings.TrimSpace(string(dir))
	file := tmp + "/install.sh"
	err = u.M.Pass(ctx, false, "curl", "-fsSL", "--retry", "3", "--proto-redir", "=https", "-o", file, u.URL+"/download/"+tag+"/install.sh")
	if err == nil {
		u.Out.Say("%s: installing ws %s", macUpdateStage, tag)
		err = u.M.Pass(ctx, false, "env", "WS_VERSION="+tag, "WS_RELEASES_URL="+u.URL, "sh", file)
	}
	_, _ = u.M.Run(ctx, "rm", "-rf", tmp)
	if err != nil {
		return fmt.Errorf("installing ws %s: %w", tag, err)
	}
	return u.M.Exec(u.Home+"/.local/bin/ws", u.Args, nil)
}
