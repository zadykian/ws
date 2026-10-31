package wizard

import (
	"context"
	"strings"
)

// send sends the Mac's clone to ws, as a git bundle of its origin/main after a fetch, where the
// files script's bundle op names a path: ws lacks its own clone. The fetch takes gh's login for
// github.com, whatever git's config names, and never prompts. The bundle goes by an ssh of its own,
// as the files script is on the first one's stdin.
func (h *HookWs) send(ctx context.Context, hook string) error {
	out, err := h.Files.Do(ctx, "bundle", hook)
	pack := strings.TrimSpace(string(out))
	if err != nil || pack == "" {
		return err
	}
	gh, _, err := h.Mac.Gh.ghPath(ctx)
	if err != nil {
		return err
	}
	h.Out.Say("%s: sending the Mac's clone of %s to ws", hookWsStage, hook)
	dir := h.Mac.Home + "/repository/" + hook[strings.Index(hook, "/")+1:]
	file := h.Host.Lock + "/hook.bundle"
	cred := "credential.https://github.com.helper="
	if err := h.M.Pass(ctx, false, "env", "GIT_TERMINAL_PROMPT=0", "git", "-C", dir, "-c", cred, "-c",
		cred+"!"+gh+" auth git-credential", "fetch", "-q", "origin"); err != nil {
		return err
	}
	if err := h.M.Pass(ctx, false, "git", "-C", dir, "bundle", "create", "-q", file, "origin/main"); err != nil {
		return err
	}
	bundle, err := h.M.ReadFile(file)
	if err != nil {
		return err
	}
	if _, err := h.Host.SSH.RunInput(ctx, bundle, "-o", "BatchMode=yes", "ws-ssh", "umask 077 && cat >"+pack); err != nil {
		return err
	}
	_, err = h.M.Run(ctx, "rm", "-f", file)
	return err
}

// apply runs apply.sh on ws, its stdin the item's secret, as security reads it, through hookGuard,
// or empty without an item. Where security fails, it stops, as in unread.
func (h *HookWs) apply(ctx context.Context, hook, secret string) error {
	script := "/root/repository/" + hook[strings.Index(hook, "/")+1:] + "/apply.sh"
	args := []string{"-o", "BatchMode=yes", "ws-ssh", script}
	var in []byte
	if secret != "" {
		var err error
		if in, err = h.M.Run(ctx, "security", "find-generic-password", "-s", secret, "-w"); err != nil {
			return h.unread(secret, err)
		}
		args = []string{"-o", "BatchMode=yes", "ws-ssh", "sh -c '" + hookGuard + "'", script}
	}
	return h.Host.SSH.PassInput(ctx, in, args...)
}
