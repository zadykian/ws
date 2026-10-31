package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// hookWsStage names the stage in its messages.
const hookWsStage = "Hook on ws"

// HookWs is the stage that runs the apply.sh of WS_HOOK's repository as root on ws, by plain ssh,
// never ssh -t, at which such a script could prompt and echo its stdin. Where ws lacks
// /root/repository/REPO, the Mac's clone goes there first, as a git bundle of origin/main, which the
// files script clones on branch main, with origin on GitHub. Root's gh on ws is never used, and a
// clone there is never pulled. apply.sh's stdin is the Keychain item that WS_HOOK_SECRET names, or
// else empty. A stamp on ws marks a run of it that exited 0, so that a second run leaves it alone.
type HookWs struct {
	Host *HostKey
	// Run is ws's run stage, which installs git on ws. Mac is the clone's stage.
	Run *WsRun
	Mac *HookMac
	// Files runs the files script's operations as root on ws.
	Files *Remote
	// User is the Mac's user, whom the Keychain item belongs to.
	User string
	Tty  bool
	M    effect.Machine
	Out  effect.Out
	// ran is set once the stage ran apply.sh in this run.
	ran bool
}

// Name is the stage's name.
func (*HookWs) Name() string { return hookWsStage }

// WaitsOn is Hook on the Mac, whose clone it sends. ws's run, which installs git on ws, comes first
// in Plan.
func (*HookWs) WaitsOn() string { return hookMacStage }

// hookWsItem is the change that adds the Keychain item that WS_HOOK_SECRET names, by security.
type hookWsItem struct{ cmd string }

func (c hookWsItem) String() string {
	return hookWsStage + ": would run " + c.cmd + ", which asks for the item's secret at the terminal"
}

type hookWsRun struct{ hook, dir, in string }

func (c hookWsRun) String() string {
	return hookWsStage + ": would send the Mac's clone of " + c.hook + " to " + c.dir + " on ws, where " +
		"missing, then run its apply.sh there, its stdin " + c.in
}

// item returns the command that adds the Keychain item, or an empty string where the Keychain has it.
func (h *HookWs) item(ctx context.Context, secret string) string {
	if secret == "" {
		return ""
	}
	if _, err := h.M.Run(ctx, "security", "find-generic-password", "-s", secret); err == nil {
		return ""
	}
	return "security add-generic-password -a " + h.User + " -s " + secret + " -w"
}

// Plan is empty where mac.conf names no repository, or where ws has the stamp of its apply.sh, which
// names it as OWNER/REPO. Under --update, only once the stage ran apply.sh in this run.
func (h *HookWs) Plan(ctx context.Context) (step.Plan, error) {
	c, err := h.Host.conf()
	if err != nil || c.Hook == "" {
		return step.Plan{}, err
	}
	if h.Host.Check && !h.Run.runDone(ctx, c.Target) {
		h.Out.Say("%s: after %s", hookWsStage, runStage)
		return step.Plan{}, nil
	}
	if (!h.Run.Update || h.ran) && h.Run.Remote.Status(ctx) &&
		strings.Contains(" "+h.Run.Remote.Get("hook")+" ", " "+c.Hook+" ") {
		return step.Plan{}, nil
	}
	in := "empty"
	if c.HookSecret != "" {
		in = "the Keychain item " + c.HookSecret
	}
	var cs []step.Change
	if cmd := h.item(ctx, c.HookSecret); cmd != "" {
		cs = append(cs, hookWsItem{cmd})
	}
	dir := "/root/repository/" + c.Hook[strings.Index(c.Hook, "/")+1:]
	return step.Plan{Changes: append(cs, hookWsRun{c.Hook, dir, in})}, nil
}
