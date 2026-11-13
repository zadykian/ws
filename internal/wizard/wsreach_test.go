package wizard

import (
	"context"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// On ws, the login runs by runuser as host_vars' WS_USER, and a missing claude names ws server
// ubuntu apply.
func TestWsClaudeHere(t *testing.T) {
	ws, m := sessiontest.New(session.Status{Claude: "none"})
	c := &stages.WsClaude{Ws: ws, Tty: true}
	if _, err := c.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "run sudo ws server ubuntu apply") {
		t.Errorf("Plan where claude is missing = %v", err)
	}
	m.Results["runuser -l ada -c claude auth login"] = effecttest.Result{}
	if err := c.Apply(context.Background(), step.Plan{}); err != nil ||
		!slices.Contains(m.Calls, "foreground runuser -l ada -c claude auth login") {
		t.Errorf("Apply = %v; calls %q", err, m.Calls)
	}
}
