package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/update"
)

// CheckInstalled refuses where update.File, the ws that the units, the plugin and the frame's
// checks run, does not print version, this ws's. Setup readies the tree for this ws, and a ws of
// another version would read it. The error names the fix: ws update for a release, which installs
// it there and runs its setup, or an install of this build for a trial's.
func CheckInstalled(ctx context.Context, m effect.Machine, version string) error {
	fix := "install this build there, as .tools/install-build does"
	if check.Release(version) {
		fix = "run ws update --version " + strings.TrimPrefix(version, "v") +
			", which installs this release there and runs its setup"
	}
	out, err := m.Run(ctx, update.File, "version")
	if err != nil {
		return fmt.Errorf("%s, which the units run, does not run: %w: %s", update.File, err, fix)
	}
	if got := strings.TrimSpace(string(out)); got != version {
		return fmt.Errorf("this ws is %s, and %s, which the units run, is %s: %s", version, update.File, got, fix)
	}
	return nil
}
