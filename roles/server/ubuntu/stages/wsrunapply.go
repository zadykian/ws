package stages

import (
	"context"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
)

// Apply runs ws server ubuntu apply in a unit of its own through the remote script, or follows the
// run that goes on there, with its log on stderr. On a cloud VM the outbound table would drop the
// provider's metadata service. So host_vars names security_outbound: false before the run, as ws
// server ubuntu apply reads its values from that file. A unit that the run followed may have begun
// before a change to that file: a run of its own follows then.
func (r *WsRun) Apply(ctx context.Context, p step.Plan) error {
	c, ok := p.Changes[0].(runChange)
	if !ok {
		return fmt.Errorf("%s: unknown change %T", RunStage, p.Changes[0])
	}
	if c.user != "" {
		if err := r.Ws.NameUser(ctx, c.user); err != nil {
			return fmt.Errorf("%s: ws's host_vars lacks ws_user: %s: %s; mend that, then run the wizard again",
				RunStage, c.user, remote.Stderr(err))
		}
	}
	// A part's plan reads no status, where the outbound line shows.
	if c.cloud {
		if s, ok := r.Ws.Status(ctx); !ok || !s.OutboundOff {
			if err := r.outbound(ctx); err != nil {
				return err
			}
		}
	}
	r.Out.Say("%s: ws server ubuntu apply runs on ws in a unit of its own, which goes on where ssh drops; "+
		"a first run takes 20 to 40 minutes", RunStage)
	if err := r.run(ctx); err != nil {
		return err
	}
	r.ran = true
	// A part leaves the full run's stamp unchanged, which would read as a unit begun too soon.
	if len(r.Only) > 0 {
		return nil
	}
	if s, ok := r.Ws.Status(ctx); !ok {
		return fail()
	} else if !sameVars(s) {
		r.Out.Say("%s: the unit it followed began before ws's host_vars took its last change, "+
			"so ws server ubuntu apply runs again", RunStage)
		if err := r.run(ctx); err != nil {
			return err
		}
		if _, ok := r.Ws.Status(ctx); !ok {
			return fail()
		}
	}
	return nil
}

// outbound names security_outbound: false in ws's host_vars by the remote script's outbound op, and
// stops with its error where that fails.
func (r *WsRun) outbound(ctx context.Context) error {
	if err := r.Ws.Outbound(ctx); err != nil {
		lines := strings.Split(strings.TrimSpace(err.Error()), "\n")
		return fmt.Errorf("%s: ws's host_vars lacks security_outbound: false: %s; mend that, or run the "+
			"wizard with --update, which runs ws server ubuntu apply again", RunStage, lines[len(lines)-1])
	}
	return nil
}
