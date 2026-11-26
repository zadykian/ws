package cli

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/setup"
)

// bootstrapDiffers asks mise's status for what its plan leaves out: the preferences, the Dock's
// apps and the login items. --missing exits 1 for a difference and for an error alike, so --json
// tells them apart: a difference prints its report first, and an error prints none.
func bootstrapDiffers(ctx context.Context, m effect.Machine, out effect.Out) (bool, error) {
	stdout, err := m.Run(ctx, setup.MisePath, "-C", macosRoles, "-E", appsEnvs, "bootstrap", "status", "--missing", "--json")
	if err == nil {
		return false, nil
	}
	var exit interface{ ExitCode() int }
	var report map[string]json.RawMessage
	if !errors.As(err, &exit) || exit.ExitCode() != 1 || json.Unmarshal(stdout, &report) != nil {
		return false, err
	}
	out.Say("mise's status differs from the config, which its plan does not cover:")
	// The table names what differs. Its exit is the difference again.
	_ = runMise(ctx, m, "-E", appsEnvs, "bootstrap", "status", "--missing")
	return true, nil
}
