package wizard

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/step"
)

// loginStop is the change that stops the run with what to do: the stage adds no key.
type loginStop struct{}

func (loginStop) String() string {
	return loginStage + ": would stop, as root on ws refuses the Mac's key"
}

// Plan is empty where root logs in. Under --check it shows that a run would stop.
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

// Apply stops, naming the Mac's key and what gives root that key without the wizard. Where ws
// doesn't answer, the Host key stage's question gives its address now, and root may take the key
// there.
func (r *Login) Apply(ctx context.Context, _ step.Plan) error {
	w, err := r.Host.SSH.Way(ctx)
	if err != nil {
		return errors.New(loginStage + ": " + err.Error())
	}
	return r.stop(ctx, w)
}

func (r *Login) stop(ctx context.Context, w Way) error {
	if w.User != "root" {
		return fmt.Errorf("%s: ssh logs in to ws as %s, mac.conf's WS_USER, and the wizard sets ws up as root alone: "+
			"take the README's steps by hand", loginStage, w.User)
	}
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
