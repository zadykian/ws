package wizard

import (
	"context"
	"strings"
)

// runStage names the stage in its messages.
const runStage = "ws's run"

// WsRun is the stage of ws's playbook, bootstrap.sh, in a unit of its own on ws. The remote script
// starts it and follows it. A run goes on where ssh drops, and the next run of the wizard follows it.
type WsRun struct {
	// Host is the Host key stage, whose ssh and mac.conf this one uses. Check is its.
	Host   *HostKey
	Remote *Remote
	// Update has the stage run the playbook again, after the remote script pulls ws's clone.
	Update bool
	// ran is set once the stage ran the playbook in this run.
	ran bool
}

// Name is the stage's name.
func (*WsRun) Name() string { return runStage }

// WaitsOn is the Root's key stage, which the remote script needs.
func (*WsRun) WaitsOn() string { return rootStage }

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

// idle tells whether ws's last full run exited 0, and no run's unit runs there now.
func (r *WsRun) idle(ctx context.Context) bool {
	return r.Remote.Status(ctx) && r.Remote.Get("active") == "-" && r.lastRun() == "0"
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
