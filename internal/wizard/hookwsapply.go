package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zadykian/ws/step"
)

// Apply adds the Keychain item, where the Keychain lacks it, then downloads the release, sends its
// build, runs it, and stamps it. The files script's install removes the old stamp, so that a run
// that fails or is cut short leaves none.
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
	h.Out.Say("%s: %s apply runs as root on ws, its stdin %s", hookWsStage, hookName(c.Hook), in)
	h.Run.Remote.Forget()
	if err := h.deploy(ctx, c.Hook, c.HookSecret); err != nil {
		var stop hookStop
		if errors.As(err, &stop) {
			return err
		}
		return fmt.Errorf("%s: the release of %s from the Mac to ws, or its command, failed, as above; mend "+
			"what they printed, then run the wizard again, which runs it again", hookWsStage, c.Hook)
	}
	h.ran = true
	return nil
}

// deploy reads the secret, downloads and checks the build, sends it, runs it and stamps it.
func (h *HookWs) deploy(ctx context.Context, hook, secret string) error {
	in, err := h.read(ctx, secret)
	if err != nil {
		return err
	}
	data, sum, err := h.release(ctx, hook)
	if err != nil {
		return err
	}
	if err := h.upload(ctx, hook, data, sum); err != nil {
		return err
	}
	if err := h.apply(ctx, hook, secret, in, nil); err != nil {
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
	h.Out.Say("%s: %s asks for the secret of the command, twice", hookWsStage, cmd)
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
		"the command didn't run; unlock or allow it, then run the wizard again" + why)}
}
