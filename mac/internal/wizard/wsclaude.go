package wizard

import (
	"context"
	"errors"

	"github.com/zadykian/ws/step"
)

// claudeStage names the stage in its messages.
const claudeStage = "claude on ws"

// WsClaude is the stage that logs root's claude in on ws, in a session of ssh -t at the terminal,
// through a login shell, whose PATH has ~/.local/bin, where claude is.
type WsClaude struct {
	// Run is ws's run stage, which installs claude on ws.
	Run *WsRun
	// Tty tells whether a terminal is there to log in at.
	Tty bool
}

// Name is the stage's name.
func (*WsClaude) Name() string { return claudeStage }

// WaitsOn is ws's run.
func (*WsClaude) WaitsOn() string { return runStage }

type claudeLogin struct{}

func (claudeLogin) String() string {
	return claudeStage + ": would run claude auth login as root on ws, at the terminal"
}

// Plan is empty where root's claude on ws has a login.
func (c *WsClaude) Plan(ctx context.Context) (step.Plan, error) {
	if c.Run.Remote.Status(ctx) && c.Run.Remote.Get("claude") == "ok" {
		return step.Plan{}, nil
	}
	if c.Run.Remote.Get("claude") == "none" {
		return step.Plan{}, errors.New(claudeStage + ": ws lacks root's claude, which its run installs: " +
			"run the wizard with --update")
	}
	return step.Plan{Changes: []step.Change{claudeLogin{}}}, nil
}

// Apply runs claude auth login as root on ws. Where it fails, the stage says to run the wizard again.
func (c *WsClaude) Apply(ctx context.Context, _ step.Plan) error {
	if !c.Tty {
		return errors.New(claudeStage + ": root's claude has no login, and no terminal is there for " +
			"claude auth login: run it on ws, then the wizard again")
	}
	c.Run.Host.Out.Say("%s: claude auth login runs as root on ws: take its steps at this terminal, "+
		"and open the address it shows in the Mac's browser", claudeStage)
	c.Run.Remote.Forget()
	if err := c.Run.Host.SSH.Pass(ctx, "-t", "ws-ssh", `exec bash -lc "claude auth login"`); err != nil {
		return errors.New(claudeStage + ": root's claude on ws still has no login, as above; run the wizard again")
	}
	return nil
}
