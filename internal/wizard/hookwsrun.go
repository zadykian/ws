package wizard

import (
	"context"
	"strings"
)

// hookGuard is the script that ssh runs on ws as sh -c GUARD APPLY, where WS_HOOK_SECRET names an
// item. It runs APPLY, apply.sh, on its stdin, but exits 3 where that is empty. So apply.sh never
// runs without its secret.
const hookGuard = `IFS= read -r l || [ -n "$l" ] || exit 3; { printf %s "$l"; echo; cat; } | exec "$0"`

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

// apply runs apply.sh on ws, its stdin the item's secret, as read gave it, through hookGuard, or
// empty without an item.
func (h *HookWs) apply(ctx context.Context, hook, secret string, in []byte) error {
	repo := hookRepo(hook)
	script := "/root/repository/" + repo[strings.Index(repo, "/")+1:] + "/apply.sh"
	args := []string{"-o", "BatchMode=yes", "ws-ssh", script}
	if secret != "" {
		args = []string{"-o", "BatchMode=yes", "ws-ssh", "sh -c '" + hookGuard + "'", script}
	}
	return h.Host.SSH.PassInput(ctx, in, args...)
}
