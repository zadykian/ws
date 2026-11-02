package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/step"
)

// Apply runs ws server apply in a unit of its own through the remote script, or follows the run that
// goes on there, with its log on stderr. On a cloud VM, where the outbound table would drop the
// provider's metadata service, host_vars names security_outbound: false before the run, as ws
// server apply reads its values from that file. A unit that the run followed may have begun before
// that line: a run of its own follows then.
func (r *WsRun) Apply(ctx context.Context, p step.Plan) error {
	c, ok := p.Changes[0].(runChange)
	if !ok {
		return fmt.Errorf("%s: unknown change %T", runStage, p.Changes[0])
	}
	if c.outbound {
		return r.outbound(ctx)
	}
	if c.cloud && r.Remote.Get("outbound") != "off" {
		if err := r.outbound(ctx); err != nil {
			return err
		}
	}
	r.Host.Out.Say("%s: ws server apply runs on ws in a unit of its own, which goes on where ssh drops; "+
		"a first run takes 20 to 40 minutes", runStage)
	if err := r.Remote.Pass(ctx, "run"); err != nil {
		return fail()
	}
	r.ran = true
	if !c.cloud {
		return nil
	}
	if !r.Remote.Status(ctx) {
		return fail()
	}
	if !r.off() {
		r.Host.Out.Say("%s: the unit it followed began before host_vars named security_outbound: false, "+
			"so ws server apply runs again", runStage)
		if err := r.Remote.Pass(ctx, "run"); err != nil {
			return fail()
		}
		if !r.Remote.Status(ctx) {
			return fail()
		}
	}
	return nil
}

// outbound names security_outbound: false in ws's host_vars by the remote script's outbound op, and
// stops with its error where that fails.
func (r *WsRun) outbound(ctx context.Context) error {
	if _, err := r.Remote.Do(ctx, "outbound"); err != nil {
		lines := strings.Split(strings.TrimSpace(err.Error()), "\n")
		return fmt.Errorf("%s: ws's host_vars lacks security_outbound: false: %s; mend that, or run the "+
			"wizard with --update, which runs ws server apply again", runStage, lines[len(lines)-1])
	}
	return nil
}
