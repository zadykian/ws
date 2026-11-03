package wizard

import (
	"context"
	"strings"
)

// apply runs REPO apply on ws, with the flags, on an empty stdin, over plain ssh. The command needs
// no secret from the Mac: ws's secrets file and gh's login are ws's own.
func (h *HookWs) apply(ctx context.Context, hook string, flags []string) error {
	args := []string{"-o", "BatchMode=yes", "ws-ssh", hookBin(hook), "apply"}
	return h.Host.SSH.PassInput(ctx, nil, append(args, flags...)...)
}

// dryRun runs REPO apply --check on ws where ws holds the command of the tag, which a run before
// installed: its output says what the run would change. Where ws lacks it, nothing runs, and the
// plan says the command comes first.
func (h *HookWs) dryRun(ctx context.Context, hook string) error {
	out, err := h.Files.Do(ctx, "have", hook)
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return err
	}
	h.Out.Say("%s: %s apply --check runs as root on ws", hookWsStage, hookName(hook))
	return h.apply(ctx, hook, []string{"--check"})
}
