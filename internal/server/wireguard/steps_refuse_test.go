package wireguard

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/server/values"
)

func TestStepsStopOnTheSettings(t *testing.T) {
	v := values.Default()
	v.Wireguard.Port = 0
	ss := steps(v, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{}, bytes.NewReader(nil), nil)
	if _, err := step.Run(t.Context(), ss, &effecttest.Out{}, true); err == nil || !strings.Contains(err.Error(), "wireguard_port") {
		t.Errorf("run = %v, want the refusal", err)
	}
}

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
