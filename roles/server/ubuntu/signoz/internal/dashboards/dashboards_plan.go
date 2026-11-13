package dashboards

import (
	"context"
	"fmt"
	"net/http"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

type create struct {
	s    Dashboards
	name string
	body []byte
}

func (c create) String() string { return "create the dashboard " + c.name }

func (c create) Apply(ctx context.Context) error {
	return c.s.API.Call(ctx, http.MethodPost, "/api/v2/dashboards", c.body, http.StatusCreated, nil)
}

type replace struct {
	create
	id string
}

func (c replace) String() string { return "replace the dashboard " + c.name }

func (c replace) Apply(ctx context.Context) error {
	return c.s.API.Call(ctx, http.MethodPut, "/api/v2/dashboards/"+c.id, c.body, http.StatusOK, nil)
}

// Plan lists SigNoz's dashboards and compares each of the step's with its file. SigNoz refuses to
// change a dashboard locked in its UI: the step leaves it unchanged and says so, rather than fail
// and stop the roles after this one.
func (s Dashboards) Plan(ctx context.Context) (step.Plan, error) {
	files := make(map[string]dashboard, len(s.Names))
	bodies := make(map[string][]byte, len(s.Names))
	for _, n := range s.Names {
		d, body, err := load(n)
		if err != nil {
			return step.Plan{}, err
		}
		files[n], bodies[n] = d, body
	}
	if !host.Booted(s.M) {
		s.Out.Say("%s: skipped, systemd is not running", s.Name())
		return step.Plan{}, nil
	}
	have, err := s.list(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	var p step.Plan
	for _, n := range s.Names {
		c := create{s, n, bodies[n]}
		i := indexOf(have, n)
		if i < 0 {
			p.Changes = append(p.Changes, c)
			continue
		}
		var got struct {
			Data dashboard `json:"data"`
		}
		if err := s.API.Get(ctx, "/api/v2/dashboards/"+have[i].ID, &got); err != nil {
			return step.Plan{}, err
		}
		switch {
		case !got.Data.differs(files[n]):
		case got.Data.Locked:
			effect.SayLasting(s.Out, "The %s dashboard is locked in SigNoz's UI and differs from dashboards/%s.json: unlock it for a run to replace it.", n, n)
		default:
			p.Changes = append(p.Changes, replace{c, have[i].ID})
		}
	}
	return p, nil
}

// Apply creates and replaces what the plan found due.
func (Dashboards) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := c.(host.Change).Apply(ctx); err != nil {
			return fmt.Errorf("%s: %w", step.Headline(c), err)
		}
	}
	return nil
}
