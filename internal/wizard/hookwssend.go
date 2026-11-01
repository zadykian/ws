package wizard

import (
	"context"
	"fmt"
	"strings"
)

// send sends the Mac's clone to ws, as a git bundle of its origin/main after a fetch, where the
// files script's bundle op names a path: ws lacks its own clone. Given a tag, the bundle holds the
// tag alone, fetched without a + so that a tag moved on GitHub stops the stage, and ws lacks the
// tag at its HEAD. The fetch takes gh's login for
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
	repo := hookRepo(hook)
	dir := h.Mac.Home + "/repository/" + repo[strings.Index(repo, "/")+1:]
	file := h.Host.Lock + "/hook.bundle"
	cred := "credential.https://github.com.helper=" //nolint:gosec // a git config key, not a credential
	ref, tag := "origin/main", hookTag(hook)
	args := []string{"-q", "origin"}
	if tag != "" {
		// Without -q, git's own line says why it refused a tag: moved, absent or unreachable.
		ref = "refs/tags/" + tag
		args = []string{"origin", ref + ":" + ref}
	}
	fetch := append([]string{"GIT_TERMINAL_PROMPT=0", "git", "-C", dir, "-c", cred, "-c",
		cred + "!" + gh + " auth git-credential", "fetch"}, args...)
	if err := h.M.Pass(ctx, false, "env", fetch...); err != nil {
		if tag != "" {
			return hookStop{fmt.Errorf("%s: git cannot fetch the tag %s of %s, as above: if GitHub moved "+
				"it, use a new tag", hookWsStage, tag, repo)}
		}
		return err
	}
	if err := h.M.Pass(ctx, false, "git", "-C", dir, "bundle", "create", "-q", file, ref); err != nil {
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
