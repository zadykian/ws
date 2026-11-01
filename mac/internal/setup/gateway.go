package setup

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// The app that Homebrew's cask jetbrains-gateway puts in /Applications.
const gatewayApp = "JetBrains Gateway.app"

// Gateway installs JetBrains Gateway, through which the Mac opens Rider and GoLand on ws, from
// Homebrew's cask where it finds no app. A failed install is a warning, so that the tunnel's steps
// still run, and the next run tries again. Where Homebrew has the cask but no app is found, it
// warns instead, as brew install would upgrade the cask. Gateway updates itself, so it runs no
// brew upgrade. Without Homebrew it stops.
type Gateway struct {
	Home string
	M    effect.Machine
	Out  effect.Out
	// LookPath finds a program on the user's PATH. Nil means exec.LookPath.
	LookPath func(string) (string, error)
	tried    bool // the install ran, so the plan is empty for the rest of the run
}

// Name is gateway.
func (*Gateway) Name() string { return "gateway" }

// WaitsOn is nothing.
func (*Gateway) WaitsOn() string { return "" }

// there reports whether Gateway's app is in /Applications, where its cask puts it, or in
// ~/Applications. An app of that name installed another way counts too.
func (g *Gateway) there() bool {
	for _, dir := range []string{"/Applications", g.Home + "/Applications"} {
		if _, err := g.M.Stat(dir + "/" + gatewayApp); !errors.Is(err, fs.ErrNotExist) {
			return err == nil
		}
	}
	return false
}

// Plan finds whether the app is there, and whether Homebrew has the cask.
func (g *Gateway) Plan(ctx context.Context) (step.Plan, error) {
	if g.there() || g.tried {
		return step.Plan{}, nil
	}
	look := g.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("brew"); err != nil {
		return step.Plan{}, errors.New("install Homebrew first: https://brew.sh")
	}
	if _, err := g.M.Run(ctx, "brew", "list", "--cask", "jetbrains-gateway"); err != nil {
		return step.Plan{Changes: []step.Change{gatewayInstall}}, nil
	}
	g.warnKept()
	return step.Plan{}, nil
}

var gatewayInstall = Run{Name: "brew", Args: []string{"install", "--cask", "jetbrains-gateway"}}

func (g *Gateway) warnKept() {
	g.Out.Warn("Homebrew has the cask jetbrains-gateway, but no %s is in /Applications or "+
		"~/Applications: brew reinstall --cask jetbrains-gateway puts it in /Applications, unless "+
		"HOMEBREW_CASK_OPTS names another --appdir", gatewayApp)
}

// Apply installs the cask, and warns where the install fails or leaves no app. The brew install's
// output reaches the terminal.
func (g *Gateway) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		g.tried = true
		g.Out.Say("installing jetbrains-gateway")
		if err := Apply(ctx, g.M, g.Out, c); err != nil {
			g.Out.Warn("%s failed, so the Mac has no JetBrains Gateway: the next run tries again",
				gatewayInstall.line())
		} else if !g.there() {
			g.warnKept()
		}
	}
	return nil
}
