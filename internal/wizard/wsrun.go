package wizard

import (
	"context"
	"errors"
	"strings"

	"github.com/zadykian/ws/step"
)

// runStage names the stage in its messages.
const runStage = "ws's run"

// WsRun is the stage of ws's run, ws server apply, in a unit of its own on ws. The remote script
// starts it and follows it. A run goes on where ssh drops, and the next run of the wizard follows it.
type WsRun struct {
	// Host is the Host key stage, whose ssh and mac.conf this one uses. Check is its.
	Host   *HostKey
	Remote *Remote
	// Files is the files script, which names the tunnel's peer.
	Files *Remote
	// Version is this ws's version. A full run counts only where it was made by a ws of that version.
	Version string
	// Secrets is the Mac's secrets file and User the Mac's user, whose Keychain holds its identity. A
	// run sends the file and the identity, and the run's stamp names the file's SHA-256.
	Secrets, User string
	// Update has the stage run ws server apply once more, with the ws installed there.
	Update bool
	// ran is set once the stage ran ws server apply in this run.
	ran bool
}

// Name is the stage's name.
func (*WsRun) Name() string { return runStage }

// WaitsOn is the stage that installs this ws on ws.
func (*WsRun) WaitsOn() string { return installStage }

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

// idle tells whether ws's last full run exited 0, made by a ws of this version, with the Mac's
// secrets file as it is now, and no run's unit runs there now.
func (r *WsRun) idle(ctx context.Context) bool {
	sum, err := r.localSum()
	return err == nil && r.Remote.Status(ctx) && r.Remote.Get("active") == "-" && r.lastRun() == "0" &&
		strings.HasPrefix(r.Remote.Get("run"), r.Version+" ") && r.stampSum() == sum
}

// runDone succeeds where ws's last full run exited 0 and no run's unit runs there. On a cloud VM,
// that run must have begun with host_vars turning the outbound table off, and ws's host_vars must keep it off. Under
// --update, only once the stage ran ws server apply in this run.
func (r *WsRun) runDone(ctx context.Context, target string) bool {
	if r.Update && !r.ran {
		return false
	}
	return r.idle(ctx) && (target != "cloud" || (r.off() && r.Remote.Get("outbound") == "off"))
}

// runChange is the change that runs ws server apply, or that names the outbound table's state in host_vars.
type runChange struct {
	text string
	// outbound has the stage name the state alone.
	outbound bool
	// cloud is set for a cloud VM, whose run turns the outbound table off.
	cloud bool
}

func (c runChange) String() string { return runStage + ": would " + c.text }

// Plan reads ws's last run, and finds whether ws server apply is to run, to be followed, or whether
// host_vars alone lacks the outbound line. Under --check it says what would happen.
func (r *WsRun) Plan(ctx context.Context) (step.Plan, error) {
	c, err := r.Host.conf()
	if err != nil {
		return step.Plan{}, err
	}
	if _, err := r.localSum(); err != nil {
		return step.Plan{}, err
	}
	if r.runDone(ctx, c.Target) {
		return step.Plan{}, nil
	}
	how := "run ws server apply on ws in a unit of its own, and follow its log"
	if data, _ := r.macSecrets(); data != nil {
		how += ", after sending the secrets file and the identity"
	}
	if c.Target == "cloud" {
		how += ", after naming security_outbound: false in ws's host_vars"
		if !r.Update && r.idle(ctx) && r.off() {
			return step.Plan{Changes: []step.Change{runChange{text: "name security_outbound: false in ws's host_vars",
				outbound: true}}}, nil
		}
	}
	if r.Host.Check {
		unit, last := r.Remote.Get("active"), r.lastRun()
		if unit != "" && unit != "-" {
			how = "follow " + unit + ", which runs on ws"
		} else if last != "-" && last != "0" {
			how += "; its last full run exited " + last
		}
	}
	return step.Plan{Changes: []step.Change{runChange{text: how, cloud: c.Target == "cloud"}}}, nil
}

// fail says what to do where ws server apply ran no full run to its end. It names no cause, as ssh has
// printed it, or ws has.
func fail() error {
	return errors.New(runStage + ": ws server apply stopped, as above, or ssh dropped while " +
		"it goes on; mend what it printed, then run the wizard again, which follows a run that goes on")
}
