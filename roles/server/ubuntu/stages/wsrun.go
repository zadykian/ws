package stages

import (
	"context"
	"errors"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// RunStage names the stage in its messages.
const RunStage = "ws's run"

// WsRun is the stage of ws's run, ws server ubuntu apply, in a unit of its own on ws. The remote
// script starts it and follows it. A run goes on where ssh drops, and the next run of the wizard
// follows it. The Mac's wizard and ws's both run it, each on its own session.
type WsRun struct {
	// Ws is the session to ws, whose files script names ws_user and the tunnel's peer.
	Ws *session.Session
	// Await names the stage that must be done first. A run needs one: the Mac's list names the stage
	// that installs this ws on ws, and ws's own names its values.
	Await string
	// Unbooted is set on ws where its systemd isn't booted, as in a container: no unit runs there, so
	// the stage says so and skips.
	Unbooted bool
	// Check is set under --check. Out takes the stage's lines.
	Check bool
	Out   effect.Out
	// Version is this ws's version. A full run counts only where a ws of that version made it.
	Version string
	// Update has the stage run ws server ubuntu apply once more, with the ws installed there.
	Update bool
	// Only is what a run of part of ws takes, by --only's entries. The remote script stamps a full run
	// alone, so such a run leaves the stage due as it found it.
	Only []string
	// ran is set once the stage ran ws server ubuntu apply in this run.
	ran bool
}

// Name is the stage's name.
func (*WsRun) Name() string { return RunStage }

// Side is ws: ws server ubuntu apply runs there.
func (*WsRun) Side() guided.Side { return session.Ws }

// WaitsOn is Await.
func (r *WsRun) WaitsOn() string { return r.Await }

// RunDone succeeds where ws's last full run left what c wants. Under --update, only once the stage
// ran ws server ubuntu apply in this run.
func (r *WsRun) RunDone(ctx context.Context, c session.Want) bool {
	if r.Update && !r.ran {
		return false
	}
	s, ok := r.Ws.Status(ctx)
	return ok && done(s, r.Version, c)
}

// runChange is the change that runs ws server ubuntu apply.
type runChange struct {
	text string
	// cloud is set for a cloud VM, whose run turns the outbound table off.
	cloud bool
	// user is the ws_user that host_vars is to name before the run, where it names another.
	user string
	// pairs are the host_vars pairs for wg0, which the run writes first.
	pairs []string
}

func (c runChange) String() string { return RunStage + ": would " + c.text }

// Plan reads ws's last run, and finds whether ws server ubuntu apply is to run or to be followed.
// Under --check it says what would happen.
func (r *WsRun) Plan(ctx context.Context) (step.Plan, error) {
	if r.Unbooted {
		r.Out.Say("%s: ws's systemd isn't booted, as in a container, so no unit runs ws server ubuntu apply: "+
			"run sudo ws server ubuntu apply", RunStage)
		return step.Plan{}, nil
	}
	c, err := r.Ws.Want()
	if err != nil {
		return step.Plan{}, err
	}
	pairs, err := tunnelPairs(ctx, r.Ws, c)
	if err != nil {
		return step.Plan{}, err
	}
	if len(pairs) == 0 && r.partDone(ctx, c) {
		return step.Plan{}, nil
	}
	how := "run ws server ubuntu apply on ws in a unit of its own, and follow its log"
	if len(r.Only) > 0 {
		how = "run ws server ubuntu apply on ws for the steps chosen, in a unit of its own, and follow its log"
	}
	s := r.Ws.Known()
	if s.Secrets != "" {
		how += ", with ws's secrets file"
	}
	user := ""
	if !named(s, c) {
		user = c.User
		how += ", after naming ws_user: " + user + " in ws's host_vars"
	}
	if len(pairs) > 0 {
		how += ", after naming " + naming(pairs) + " in ws's host_vars"
	}
	if c.Target == "cloud" {
		how += ", after naming security_outbound: false in ws's host_vars"
	}
	if r.Check {
		unit, last := s.Active, s.Run.Exit
		if unit != "" {
			how = "follow " + unit + ", which runs on ws"
		} else if last != "" && last != "0" {
			how += "; its last full run exited " + last
		}
	}
	return step.Plan{Changes: []step.Change{runChange{text: how, cloud: c.Target == "cloud", user: user, pairs: pairs}}}, nil
}

// partDone is RunDone for what this run takes. A part that ran is all there is to do, and a part
// chosen is to run whatever the last full run left, as a step can drift after it.
func (r *WsRun) partDone(ctx context.Context, c session.Want) bool {
	if len(r.Only) > 0 {
		return r.ran
	}
	return r.RunDone(ctx, c)
}

// fail says what to do where ws server ubuntu apply ran no full run to its end. It names no cause,
// as ssh has printed it, or ws has.
func fail() error {
	return errors.New(RunStage + ": ws server ubuntu apply stopped, as above, or ssh dropped while " +
		"it goes on; mend what it printed, then run the wizard again, which follows a run that goes on")
}
