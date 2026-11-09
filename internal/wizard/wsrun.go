package wizard

import (
	"cmp"
	"context"
	"errors"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/step"
)

// runStage names the stage in its messages.
const runStage = "ws's run"

// WsRun is the stage of ws's run, ws server apply, in a unit of its own on ws. The remote script
// starts it and follows it. A run goes on where ssh drops, and the next run of the wizard follows it.
// The Mac's wizard and ws's both run it, each filling the fields that it reads.
type WsRun struct {
	// Host is the Mac's Host key stage, whose ssh, lock and mac.conf the Mac's later stages and its
	// closing list reach through this one.
	Host *HostKey
	// Want reads the target and the WS_USER that ws's run is to leave: mac.conf's on the Mac, and
	// host_vars' on ws.
	Want func() (conf.Conf, error)
	// Await names the stage that must be done first: by default the one that installs this ws on ws.
	Await string
	// Unbooted is set on ws where its systemd isn't booted, as in a container: no unit runs there, so
	// the stage says so and skips.
	Unbooted bool
	// Check is set under --check. Out takes the stage's lines.
	Check  bool
	Out    effect.Out
	Remote *Remote
	// Files is the files script, which names ws_user and the tunnel's peer.
	Files *Remote
	// Default is the server's default ws_user, what a host_vars with no such line gives.
	Default string
	// Version is this ws's version. A full run counts only where a ws of that version made it.
	Version string
	// Update has the stage run ws server apply once more, with the ws installed there.
	Update bool
	// Only is what a run of part of ws takes, by --only's entries. The remote script stamps a full run
	// alone, so such a run leaves the stage due as it found it.
	Only []string
	// ran is set once the stage ran ws server apply in this run.
	ran bool
}

// Name is the stage's name.
func (*WsRun) Name() string { return runStage }

// WaitsOn is Await, or the stage that installs this ws on ws.
func (r *WsRun) WaitsOn() string { return cmp.Or(r.Await, installStage) }

// lastRun returns the status of ws's last full run, as the remote script's status read it, or -.
func (r *WsRun) lastRun() string {
	if f := strings.Fields(r.Remote.Get("run")); len(f) > 1 {
		return f[1]
	}
	return "-"
}

// off tells whether ws's last full run began with host_vars turning the outbound table off.
func (r *WsRun) off() bool {
	f := strings.Fields(r.Remote.Get("run"))
	return len(f) > 2 && f[2] == "off"
}

// idle tells whether ws's last full run, made by a ws of this version, exited 0. The run must have
// read ws's secrets file and host_vars as they are now, and no run's unit may run there now.
func (r *WsRun) idle(ctx context.Context) bool {
	return r.Remote.Status(ctx) && r.Remote.Get("active") == "-" && r.lastRun() == "0" &&
		strings.HasPrefix(r.Remote.Get("run"), r.Version+" ") && r.sameSecrets() && r.sameVars()
}

// runDone succeeds where ws's last full run exited 0 and no run's unit runs there. On a cloud VM,
// that run must have begun with host_vars turning the outbound table off, which host_vars, the same
// since, still does. Under --update, only once the stage ran ws server apply in this run.
func (r *WsRun) runDone(ctx context.Context, c conf.Conf) bool {
	if r.Update && !r.ran {
		return false
	}
	return r.idle(ctx) && r.named(c) && (c.Target != "cloud" || r.off())
}

// named tells whether ws's values give ws_user the user that ssh logs in as. It compares with the
// server's default, so that a default that moves leaves a WS_USER of the old one named in host_vars.
func (r *WsRun) named(c conf.Conf) bool {
	user := r.Remote.Get("user")
	if user == "" || user == "-" {
		user = r.Default
	}
	return user == c.User
}

// runChange is the change that runs ws server apply.
type runChange struct {
	text string
	// cloud is set for a cloud VM, whose run turns the outbound table off.
	cloud bool
	// user is the ws_user that host_vars is to name before the run, where it names another.
	user string
}

func (c runChange) String() string { return runStage + ": would " + c.text }

// Plan reads ws's last run, and finds whether ws server apply is to run or to be followed. Under
// --check it says what would happen.
func (r *WsRun) Plan(ctx context.Context) (step.Plan, error) {
	if r.Unbooted {
		r.Out.Say("%s: ws's systemd isn't booted, as in a container, so no unit runs ws server apply: "+
			"run sudo ws server apply", runStage)
		return step.Plan{}, nil
	}
	c, err := r.Want()
	if err != nil {
		return step.Plan{}, err
	}
	if r.partDone(ctx, c) {
		return step.Plan{}, nil
	}
	how := "run ws server apply on ws in a unit of its own, and follow its log"
	if len(r.Only) > 0 {
		how = "run ws server apply on ws for the steps chosen, in a unit of its own, and follow its log"
	}
	if s := r.Remote.Get("secrets"); s != "" && s != noSum {
		how += ", with ws's secrets file"
	}
	user := ""
	if !r.named(c) {
		user = c.User
		how += ", after naming ws_user: " + user + " in ws's host_vars"
	}
	if c.Target == "cloud" {
		how += ", after naming security_outbound: false in ws's host_vars"
	}
	if r.Check {
		unit, last := r.Remote.Get("active"), r.lastRun()
		if unit != "" && unit != "-" {
			how = "follow " + unit + ", which runs on ws"
		} else if last != "-" && last != "0" {
			how += "; its last full run exited " + last
		}
	}
	return step.Plan{Changes: []step.Change{runChange{text: how, cloud: c.Target == "cloud", user: user}}}, nil
}

// partDone is runDone for what this run takes. A part that ran is all there is to do, and a part
// chosen is to run whatever the last full run left, as a step can drift after it.
func (r *WsRun) partDone(ctx context.Context, c conf.Conf) bool {
	if len(r.Only) > 0 {
		return r.ran
	}
	return r.runDone(ctx, c)
}

// fail says what to do where ws server apply ran no full run to its end. It names no cause, as ssh has
// printed it, or ws has.
func fail() error {
	return errors.New(runStage + ": ws server apply stopped, as above, or ssh dropped while " +
		"it goes on; mend what it printed, then run the wizard again, which follows a run that goes on")
}
