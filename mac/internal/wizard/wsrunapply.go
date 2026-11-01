package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/step"
)

// Apply runs ws's playbook in a unit of its own through the remote script, or follows the run that
// goes on there, with its log on stderr. On a cloud VM, where the outbound table would drop the
// provider's metadata service, the run's -e turns it off, and its unit names that in host_vars for
// the runs after. A unit that the run followed may have lacked that -e: a run of its own follows
// then. Where the line is missing still, outbound writes it.
func (r *WsRun) Apply(ctx context.Context, p step.Plan) error {
	c, ok := p.Changes[0].(runChange)
	if !ok {
		return fmt.Errorf("%s: unknown change %T", runStage, p.Changes[0])
	}
	if c.outbound {
		return r.outbound(ctx)
	}
	r.Host.Out.Say("%s: bootstrap.sh runs on ws in a unit of its own, which goes on where ssh drops; "+
		"a first run takes 20 to 40 minutes", runStage)
	if err := r.Remote.Pass(ctx, append([]string{"run"}, c.args...)...); err != nil {
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
		r.Host.Out.Say("%s: the unit it followed ran without -e security_outbound=false, so bootstrap.sh "+
			"runs again with it", runStage)
		if err := r.Remote.Pass(ctx, append([]string{"run"}, c.args...)...); err != nil {
			return fail()
		}
		if !r.Remote.Status(ctx) {
			return fail()
		}
	}
	if r.Remote.Get("outbound") != "off" {
		return r.outbound(ctx)
	}
	return nil
}

// outbound names security_outbound: false in ws's host_vars by the remote script's outbound op, and
// stops with its error where that fails. --update runs the playbook, which makes a clone gone.
func (r *WsRun) outbound(ctx context.Context) error {
	if _, err := r.Remote.Do(ctx, "outbound"); err != nil {
		lines := strings.Split(strings.TrimSpace(err.Error()), "\n")
		return fmt.Errorf("%s: ws's host_vars lacks security_outbound: false: %s; mend that, or run the "+
			"wizard with --update, which runs ws's playbook again", runStage, lines[len(lines)-1])
	}
	return nil
}
