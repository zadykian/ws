package wizard

import (
	"context"
	"errors"
	guided "github.com/zadykian/ws/engine/core/wizard"

	"github.com/zadykian/ws/engine/core/step"
)

// claudeStage names the stage in its messages.
const claudeStage = "claude on ws"

// WsClaude is the stage that logs WS_USER's claude in on ws, at the terminal, by the session's way:
// ssh -t from the Mac, runuser on ws itself.
type WsClaude struct {
	// Ws is the session to ws, whose run installs claude there.
	Ws *Session
	// Tty tells whether a terminal is there to log in at.
	Tty bool
}

// Name is the stage's name.
func (*WsClaude) Name() string { return claudeStage }

// Side is ws: the claude login of WS_USER.
func (*WsClaude) Side() guided.Side { return OnWs }

// WaitsOn is ws's run.
func (*WsClaude) WaitsOn() string { return runStage }

type claudeLogin struct{}

func (claudeLogin) String() string {
	return claudeStage + ": would run claude auth login as WS_USER on ws, at the terminal"
}

// Plan is empty where WS_USER's claude on ws has a login.
func (c *WsClaude) Plan(ctx context.Context) (step.Plan, error) {
	s, ok := c.Ws.status(ctx)
	if ok && s.Claude == "ok" {
		return step.Plan{}, nil
	}
	if s.Claude == "none" {
		return step.Plan{}, errors.New(claudeStage + ": ws lacks WS_USER's claude, which its run installs: " +
			c.Ws.rerun())
	}
	return step.Plan{Changes: []step.Change{claudeLogin{}}}, nil
}

// Apply runs claude auth login as WS_USER on ws. Where it fails, the stage says to run the wizard again.
func (c *WsClaude) Apply(ctx context.Context, _ step.Plan) error {
	if !c.Tty {
		return errors.New(claudeStage + ": WS_USER's claude has no login, and no terminal is there for " +
			"claude auth login: run it on ws, then the wizard again")
	}
	return c.Ws.claude(ctx)
}
