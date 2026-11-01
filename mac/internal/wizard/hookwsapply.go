package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/step"
)

// hookGuard is the script that ssh runs on ws as sh -c GUARD APPLY, where WS_HOOK_SECRET names an
// item. It runs APPLY, apply.sh, on its stdin, but exits 3 where that is empty. So apply.sh never
// runs without its secret.
const hookGuard = `IFS= read -r l || [ -n "$l" ] || exit 3; { printf %s "$l"; echo; cat; } | exec "$0"`

// Apply adds the Keychain item, where the Keychain lacks it, then sends the Mac's clone where ws
// lacks its own, runs apply.sh, and stamps it. The files script's clone op removes the old stamp, so
// that a run that fails or is cut short leaves none.
func (h *HookWs) Apply(ctx context.Context, p step.Plan) error {
	c, err := h.Host.conf()
	if err != nil {
		return err
	}
	for _, ch := range p.Changes {
		if add, ok := ch.(hookWsItem); ok {
			if err := h.addItem(ctx, add.cmd); err != nil {
				return err
			}
		}
	}
	in := "empty"
	if c.HookSecret != "" {
		in = "the Keychain item " + c.HookSecret
	}
	h.Out.Say("%s: %s's apply.sh runs as root on ws, its stdin %s", hookWsStage, c.Hook, in)
	h.Run.Remote.Forget()
	if err := h.deploy(ctx, c.Hook, c.HookSecret); err != nil {
		var stop hookStop
		if errors.As(err, &stop) {
			return err
		}
		return fmt.Errorf("%s: the clone of %s from the Mac to ws, or its apply.sh, failed, as above; mend "+
			"what they printed, then run the wizard again, which runs apply.sh again", hookWsStage, c.Hook)
	}
	h.ran = true
	return nil
}

// deploy reads the secret, sends the clone, clones it on ws, runs apply.sh and stamps it.
func (h *HookWs) deploy(ctx context.Context, hook, secret string) error {
	in, err := h.read(ctx, secret)
	if err != nil {
		return err
	}
	if err := h.send(ctx, hook); err != nil {
		return err
	}
	if err := h.Files.Pass(ctx, "clone", hookRepo(hook)); err != nil {
		return err
	}
	if err := h.apply(ctx, hook, secret, in); err != nil {
		return err
	}
	return h.Files.Pass(ctx, "stamp", hook)
}

// hookStop is an error that says what to do, which Apply passes on as it is.
type hookStop struct{ error }

// addItem adds the Keychain item by security, which asks for its secret twice at the terminal. The
// wizard never reads the secret.
func (h *HookWs) addItem(ctx context.Context, cmd string) error {
	if !h.Tty {
		return fmt.Errorf("%s: the Keychain lacks the item, and no terminal is there for %s: run it, then "+
			"the wizard again", hookWsStage, cmd)
	}
	h.Out.Say("%s: %s asks for the secret of the apply.sh, twice", hookWsStage, cmd)
	f := strings.Fields(cmd)
	if err := h.M.Pass(ctx, false, f[0], f[1:]...); err != nil {
		return fmt.Errorf("%s: %s failed, as above; run the wizard again", hookWsStage, cmd)
	}
	return nil
}

// unread says that security can't read the item, as in a locked Keychain, or where the user denied
// macOS's prompt for it, with security's last line of error.
func (h *HookWs) unread(secret string, err error) error {
	why := ""
	if lines := strings.Split(strings.TrimSpace(exitStatus.ReplaceAllString(err.Error(), "")), "\n"); lines[0] != "" {
		why = ": " + lines[len(lines)-1]
	}
	return hookStop{errors.New(hookWsStage + ": security can't read the item " + secret + " from the Keychain, so " +
		"apply.sh didn't run; unlock or allow it, then run the wizard again" + why)}
}
