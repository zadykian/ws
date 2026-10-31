package setup

import (
	"context"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// cldReleases is the page of cld's releases, whose latest has the install.sh that installs cld.
const cldReleases = "https://github.com/zadykian/cld/releases"

// Cld sets up cld, for cld restore tabs, which reopens Ghostty's tabs of cld's sessions: it installs
// cld into ~/.local/bin where missing, and updates it where a newer release is out, wherever it
// runs. It warns where cld can't be installed or updated, so that the run goes on to the tunnel.
// Under Check it says that ssh's file may gain the lines that start cld's watcher, as --check
// can run neither a cld nor a Ghostty that it would install.
type Cld struct {
	Home  string
	M     effect.Machine
	Out   effect.Out
	Check bool
	tried bool // install or update ran, so the plan is empty for the rest of the run
}

// Name is cld.
func (*Cld) Name() string { return "cld" }

// WaitsOn is nothing.
func (*Cld) WaitsOn() string { return "" }

// cldInstall installs cld with the install.sh of its latest release, as the claude role does on ws.
// install.sh checks the binary against the release's cld.sha256 before it moves it into place, and
// edits no shell profile.
type cldInstall struct{}

func (cldInstall) String() string {
	return "would install cld into ~/.local/bin with the install.sh of its latest release"
}

// cldUpdate has cld replace itself where a newer release is out.
type cldUpdate struct{}

func (cldUpdate) String() string { return "would run cld update" }

// Remark is a line that --check prints in the form of a change, which a run does not apply.
type Remark string

func (r Remark) String() string { return string(r) }

func (c *Cld) bin() string { return c.Home + "/.local/bin/cld" }

// has reports whether cld in ~/.local/bin is executable.
func (c *Cld) has() bool {
	info, err := c.M.Stat(c.bin())
	return err == nil && info.Mode&0o111 != 0
}

// ghosttyApp reports whether Ghostty's app is in /Applications, where its cask puts it, or in
// ~/Applications.
func (c *Cld) ghosttyApp() bool {
	for _, dir := range []string{"/Applications", c.Home + "/Applications"} {
		if info, err := c.M.Stat(dir + "/Ghostty.app"); err == nil && info.Mode.IsDir() {
			return true
		}
	}
	return false
}

// restoreTabs reports whether the cld in ~/.local/bin has restore tabs: cld help restore tabs fails
// in a cld without it.
func (c *Cld) restoreTabs(ctx context.Context) bool {
	_, err := c.M.Run(ctx, c.bin(), "help", "restore", "tabs")
	return err == nil
}

// gains reports whether config.d/ws would gain the lines that start cld's watcher: Ghostty's app
// is there or --check would install it, as Homebrew has not the cask, and ssh has no lines yet.
func (c *Cld) gains(ctx context.Context) bool {
	there := c.ghosttyApp()
	if there && c.has() && c.restoreTabs(ctx) {
		return false
	}
	if there {
		return true
	}
	_, err := c.M.Run(ctx, "brew", "list", "--cask", "ghostty")
	return err != nil
}

// Plan plans cld's install where it is missing, else its update. Under Check, a cld that is or will
// be there with restore tabs may bring the lines of ssh's file.
func (c *Cld) Plan(ctx context.Context) (step.Plan, error) {
	if c.tried {
		return step.Plan{}, nil
	}
	var cs []step.Change
	tabs := false
	if c.has() {
		cs, tabs = c.update(ctx)
		tabs = tabs || (c.Check && c.restoreTabs(ctx))
	} else {
		cs, tabs = []step.Change{cldInstall{}}, c.Check
	}
	if tabs && c.gains(ctx) {
		cs = append(cs, Remark("would add the lines that start cld's watcher to config.d/ws, "+
			"where that cld has restore tabs"))
	}
	return step.Plan{Changes: cs}, nil
}

// update plans the update of the cld there, and says whether the cld to come may have restore tabs.
// A release before 0.6.0 lacks cld update, so install.sh replaces it. A cld built from a clone, of
// no release's version, is left alone, as cld update refuses it. Only a run can ask cld to update,
// so --check reads the release that releases/latest leads to, as cld update does.
func (c *Cld) update(ctx context.Context) ([]step.Change, bool) {
	out, err := c.M.Run(ctx, c.bin(), "--version")
	have, ok := cldVersion(string(out))
	if err != nil || !ok {
		if err != nil || have == "" {
			c.Out.Warn("%s --version failed", c.bin())
		}
		return nil, false
	}
	if _, err := c.M.Run(ctx, c.bin(), "help", "update"); err != nil {
		return []step.Change{cldInstall{}}, c.Check
	}
	if !c.Check {
		return []step.Change{cldUpdate{}}, false
	}
	out, err = c.M.Run(ctx, "curl", "-fsSIL", "--retry", "3", "-o", "/dev/null", "-w", "%{url_effective}",
		cldReleases+"/latest")
	if err != nil {
		c.Out.Warn("cannot read cld's latest release from %s/latest", cldReleases)
		return nil, false
	}
	latest := string(out)
	if i := strings.LastIndex(latest, "/v"); i >= 0 {
		latest = latest[i+2:]
	}
	if !cldRelease.MatchString(latest) || !cldOlder(have, latest) {
		return nil, false
	}
	return []step.Change{Remark("would run cld update: cld " + have + ", the latest " + latest)}, true
}
