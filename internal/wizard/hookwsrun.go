package wizard

import (
	"context"
)

// hookGuard is the script that ssh runs on ws as sh -c GUARD COMMAND ARG..., where WS_HOOK_SECRET
// names an item. It runs the command with the arguments, on its stdin, but exits 3 where that is
// empty. So the command never runs without its secret.
const hookGuard = `IFS= read -r l || [ -n "$l" ] || exit 3; { printf %s "$l"; echo; cat; } | exec "$0" "$@"`

// read reads the item's secret by security, or nothing without an item. It runs before anything
// is sent, so that a locked Keychain, or a denial, stops with ws untouched. Where security fails,
// it stops, as in unread.
func (h *HookWs) read(ctx context.Context, secret string) ([]byte, error) {
	if secret == "" {
		return nil, nil
	}
	in, err := h.M.Run(ctx, "security", "find-generic-password", "-s", secret, "-w")
	if err != nil {
		return nil, h.unread(secret, err)
	}
	return in, nil
}

// apply runs REPO apply on ws, with the flags, its stdin the item's secret, as read gave it,
// through hookGuard, or empty without an item. It runs the installed command, over plain ssh.
func (h *HookWs) apply(ctx context.Context, hook, secret string, in []byte) error {
	bin := hookBin(hook)
	args := []string{"-o", "BatchMode=yes", "ws-ssh"}
	if secret == "" {
		args = append(args, bin, "apply")
	} else {
		args = append(args, "sh -c '"+hookGuard+"'", bin, "apply")
	}
	return h.Host.SSH.PassInput(ctx, in, args...)
}
