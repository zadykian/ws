package tunnel

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestStepsInAContainerSkipWhatNeedsSystemd(t *testing.T) {
	m, ss := fakeSteps()
	delete(m.Dirs, "/run/systemd/system")
	delete(m.Files, ufwRules)
	out := &effecttest.Out{}
	for _, run := range []string{"first", "second"} {
		changed, err := step.Run(t.Context(), ss, out, false)
		if err != nil || changed != (run == "first") {
			t.Fatalf("%s run = %v, %v", run, changed, err)
		}
	}
	for _, call := range m.Calls {
		if strings.Contains(call, "networkctl") || strings.Contains(call, "ufw") && strings.HasPrefix(call, "root") {
			t.Errorf("call %q needs systemd", call)
		}
	}
}
