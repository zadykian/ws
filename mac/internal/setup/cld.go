package setup

import (
	"context"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

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

// Plan plans cld's install where it is missing. Under Check, a cld that is or will
// be there with restore tabs may bring the lines of ssh's file.
func (c *Cld) Plan(ctx context.Context) (step.Plan, error) {
	if c.tried {
		return step.Plan{}, nil
	}
	var cs []step.Change
	tabs := false
	if !c.has() {
		cs, tabs = []step.Change{cldInstall{}}, c.Check
	}
	if tabs && c.gains(ctx) {
		cs = append(cs, Remark("would add the lines that start cld's watcher to config.d/ws, "+
			"where that cld has restore tabs"))
	}
	return step.Plan{Changes: cs}, nil
}
