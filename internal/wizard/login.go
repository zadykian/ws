package wizard

import (
	"context"
	"errors"

	"github.com/zadykian/ws/step"
)

// loginStop is the change that stops the run with what to do: the stage adds no key.
type loginStop struct{}

func (loginStop) String() string {
	return loginStage + ": would stop, as WS_USER on ws refuses the Mac's key"
}

// Plan is empty where WS_USER logs in. Under --check it shows that a run would stop.
func (r *Login) Plan(ctx context.Context) (step.Plan, error) {
	w, err := r.Host.SSH.Way(ctx)
	if err != nil {
		return step.Plan{}, errors.New(loginStage + ": " + err.Error())
	}
	if ok, _ := r.login(ctx, w); ok {
		return step.Plan{}, nil
	}
	if r.Host.Check {
		if err := r.stop(ctx, w); err != nil {
			return step.Plan{}, err
		}
	}
	return step.Plan{Changes: []step.Change{loginStop{}}}, nil
}

// Apply stops, naming the Mac's key and the user that holds it without the wizard. Where ws
// doesn't answer, the Host key stage's question gives its address now, and WS_USER may take the
// key there.
func (r *Login) Apply(ctx context.Context, _ step.Plan) error {
	w, err := r.Host.SSH.Way(ctx)
	if err != nil {
		return errors.New(loginStage + ": " + err.Error())
	}
	return r.stop(ctx, w)
}

func (r *Login) stop(ctx context.Context, w Way) error {
	c, err := r.Host.conf()
	if err != nil {
		return err
	}
	if r.Host.scan(ctx, w) == "" {
		if r.Host.Check {
			r.Host.Out.Say("%s: %s; a run would ask its address now", hostStage, r.Host.none(w))
			return nil
		}
		if w, _, err = r.Host.ask(ctx, c.Target, w); err != nil {
			return err
		}
	}
	if ok, sshErr := r.login(ctx, w); !ok {
		return r.refuse(ctx, w, sshErr)
	}
	return nil
}
