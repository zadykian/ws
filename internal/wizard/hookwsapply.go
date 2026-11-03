package wizard

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/step"
)

// Apply downloads the release, sends its build, runs it, and stamps it. The files script's install removes the old stamp, so that a run
// that fails or is cut short leaves none.
func (h *HookWs) Apply(ctx context.Context, _ step.Plan) error {
	c, err := h.Host.conf()
	if err != nil {
		return err
	}
	h.Out.Say("%s: %s apply runs as root on ws, its stdin empty", hookWsStage, hookName(c.Hook))
	h.Run.Remote.Forget()
	if err := h.deploy(ctx, c.Hook); err != nil {
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

// deploy downloads and checks the build, sends it, runs it and stamps it.
func (h *HookWs) deploy(ctx context.Context, hook string) error {
	data, sum, err := h.release(ctx, hook)
	if err != nil {
		return err
	}
	if err := h.upload(ctx, hook, data, sum); err != nil {
		return err
	}
	if err := h.apply(ctx, hook, nil); err != nil {
		return err
	}
	return h.Files.Pass(ctx, "stamp", hook)
}

// hookStop is an error that says what to do, which Apply passes on as it is.
type hookStop struct{ error }
