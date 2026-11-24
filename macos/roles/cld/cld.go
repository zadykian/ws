// Package cld installs the cld program on the Mac and keeps it at its latest release, as ws macos
// macos cld, which cld's unit runs.
package cld

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// cldReleases is the page of cld's releases, whose latest has the install.sh that installs cld.
const cldReleases = "https://github.com/zadykian/cld/releases"

// Cld sets up cld, for cld restore tabs, which reopens Ghostty's tabs of cld's sessions. It
// installs cld into ~/.local/bin where missing, and updates it where releases/latest is newer.
// Its plan reads the latest in a check too, so that the unit's check and its run agree,
// and a cld at the latest plans nothing. A failure to install, to update or to read the latest is a
// warning.
type Cld struct {
	Home string
	M    effect.Machine
	Out  effect.Out
}

// cldInstall installs cld with the install.sh of its latest release, as the claude role does on ws.
// install.sh checks the binary against the release's cld.sha256 before it moves it into place, and
// edits no shell profile.
type cldInstall struct{ c *Cld }

func (cldInstall) String() string {
	return "would install cld into ~/.local/bin with the install.sh of its latest release"
}

// cldUpdate has cld replace itself, from the release have to the latest.
type cldUpdate struct {
	c            *Cld
	have, latest string
}

func (u cldUpdate) String() string {
	return "would run cld update: cld " + u.have + ", the latest " + u.latest
}

func (cldInstall) Kind() step.ChangeKind { return step.Install }
func (cldUpdate) Kind() step.ChangeKind  { return step.Install }

func (c *Cld) bin() string { return c.Home + "/.local/bin/cld" }

// Plan returns cld's install where cld is missing, else its update where a newer release is out.
func (c *Cld) Plan(ctx context.Context) ([]resource.Change, error) {
	if info, err := c.M.Stat(c.bin()); err == nil && info.Mode&0o111 != 0 {
		return c.update(ctx), nil
	}
	return []resource.Change{cldInstall{c}}, nil
}

// update plans the update of the cld there. A release before 0.6.0 lacks cld update, so install.sh
// replaces it. A cld built from a clone, of no release's version, is left alone, as cld update
// refuses it. It reads the release that releases/latest leads to, as cld update does.
func (c *Cld) update(ctx context.Context) []resource.Change {
	out, err := c.M.Run(ctx, c.bin(), "--version")
	have, ok := cldVersion(string(out))
	if err != nil || !ok {
		if err != nil || have == "" {
			c.Out.Warn("%s --version failed", c.bin())
		}
		return nil
	}
	if _, err := c.M.Run(ctx, c.bin(), "help", "update"); err != nil {
		return []resource.Change{cldInstall{c}}
	}
	out, err = c.M.Run(ctx, "curl", "-fsSIL", "--retry", "3", "-o", "/dev/null", "-w", "%{url_effective}",
		cldReleases+"/latest")
	if err != nil {
		c.Out.Warn("cannot read cld's latest release from %s/latest", cldReleases)
		return nil
	}
	latest := string(out)
	if i := strings.LastIndex(latest, "/v"); i >= 0 {
		latest = latest[i+2:]
	}
	if !cldRelease.MatchString(latest) || !cldOlder(have, latest) {
		return nil
	}
	return []resource.Change{cldUpdate{c, have, latest}}
}
