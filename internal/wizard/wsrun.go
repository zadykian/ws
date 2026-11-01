package wizard

import (
	"context"
	"errors"
	"strings"

	"github.com/zadykian/ws/step"
)

// runStage names the stage in its messages.
const runStage = "ws's run"

// WsRun is the stage of ws's playbook, ws server apply, in a unit of its own on ws. The remote script
// starts it and follows it. A run goes on where ssh drops, and the next run of the wizard follows it.
type WsRun struct {
	// Host is the Host key stage, whose ssh and mac.conf this one uses. Check is its.
	Host   *HostKey
	Remote *Remote
	// Version is this ws's version. A full run counts only where it was made by a ws of that version.
	Version string
	// Update has the stage run the playbook once more, with the ws installed there.
	Update bool
	// ran is set once the stage ran the playbook in this run.
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

// off tells whether ws's last full run turned the outbound table off by its -e.
func (r *WsRun) off() bool {
	f := strings.Fields(r.Remote.Get("run"))
	return len(f) > 2 && f[2] == "off"
}

// idle tells whether ws's last full run exited 0, made by a ws of this version, and no run's unit
// runs there now.
func (r *WsRun) idle(ctx context.Context) bool {
	return r.Remote.Status(ctx) && r.Remote.Get("active") == "-" && r.lastRun() == "0" &&
		strings.HasPrefix(r.Remote.Get("run"), r.Version+" ")
}

// runDone succeeds where ws's last full run exited 0 and no run's unit runs there. On a cloud VM,
// that run's -e must have turned the outbound table off, and ws's host_vars must keep it off. Under
// --update, only once the stage ran the playbook in this run.
func (r *WsRun) runDone(ctx context.Context, target string) bool {
	if r.Update && !r.ran {
		return false
	}
	return r.idle(ctx) && (target != "cloud" || (r.off() && r.Remote.Get("outbound") == "off"))
}

// runChange is the change that runs the playbook, or that names the outbound table's state in host_vars.
type runChange struct {
	text string
	// args are the remote script's run arguments. outbound has the stage name the state alone.
	args     []string
	outbound bool
	// cloud is set for a cloud VM, whose run turns the outbound table off.
	cloud bool
}

func (c runChange) String() string { return runStage + ": would " + c.text }

// Plan reads ws's last run, and finds whether the playbook is to run, to be followed, or whether
// host_vars alone lacks the outbound line. Under --check it says what would happen.
func (r *WsRun) Plan(ctx context.Context) (step.Plan, error) {
	c, err := r.Host.conf()
	if err != nil {
		return step.Plan{}, err
	}
	if r.runDone(ctx, c.Target) {
		return step.Plan{}, nil
	}
	how := "run ws server apply on ws in a unit of its own, and follow its log"
	var args []string
	if c.Target == "cloud" {
		args = append(args, "-e", "security_outbound=false")
		how += ", with -e security_outbound=false, then name that in ws's host_vars"
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
	return step.Plan{Changes: []step.Change{runChange{text: how, args: args, cloud: c.Target == "cloud"}}}, nil
}

// fail says what to do where ws's playbook ran no full run to its end. It names no cause, as ssh has
// printed it, or the playbook has.
func fail() error {
	return errors.New(runStage + ": ws's playbook stopped, as above, or ssh dropped while " +
		"it goes on; mend what it printed, then run the wizard again, which follows a run that goes on")
}
