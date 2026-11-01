package setup

import (
	"context"
	"strings"

	"github.com/zadykian/ws/mac/internal/step"
)

// Apply installs or updates cld. A failure is a warning, and the next run tries again.
func (c *Cld) Apply(ctx context.Context, p step.Plan) error {
	c.tried = true
	for _, ch := range p.Changes {
		switch ch.(type) {
		case cldInstall:
			c.install(ctx)
		case cldUpdate:
			c.runUpdate(ctx)
		}
	}
	return nil
}

// install runs the install.sh of cld's latest release, from a file that is removed after, and
// passes its output to the terminal.
func (c *Cld) install(ctx context.Context) {
	out, err := c.M.Run(ctx, "mktemp", "-t", "ws-cld")
	if err != nil {
		c.Out.Warn("cannot make a file for cld's install.sh: %v", err)
		return
	}
	script := strings.TrimSpace(string(out))
	defer func() { _, _ = c.M.Run(ctx, "rm", "-f", script) }()
	if _, err := c.M.RunShown(ctx, false, "curl", "-fsSL", "--retry", "3", "--proto-redir", "=https",
		"-o", script, cldReleases+"/latest/download/install.sh"); err != nil {
		c.Out.Warn("cannot download cld's install.sh from %s/latest", cldReleases)
		return
	}
	if err := c.M.Pass(ctx, false, "env", "CLD_INSTALL_DIR="+c.Home+"/.local/bin", "sh", script); err != nil {
		c.Out.Warn("cld's install.sh failed")
	}
}

// runUpdate has cld update itself and says what it prints where it did.
func (c *Cld) runUpdate(ctx context.Context) {
	out, err := c.M.Run(ctx, "sh", "-c", `exec "$0" update 2>&1`, c.bin())
	text := strings.TrimRight(string(out), "\n")
	if err != nil {
		c.Out.Warn("cld update failed: %s", text)
	}
	if strings.HasPrefix(text, "Updated") {
		c.Out.Say("%s", text)
	}
}
