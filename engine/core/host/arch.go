package host

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// Architecture returns the machine's architecture as dpkg names it, such as amd64 or arm64, which
// the releases' assets use. Go's name would be wrong for a 32-bit userland on a 64-bit kernel.
func Architecture(ctx context.Context, m effect.Machine) (string, error) {
	out, err := m.Run(ctx, "dpkg", "--print-architecture")
	if err != nil {
		return "", fmt.Errorf("reading dpkg's architecture: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
