package signoz

import (
	"regexp"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
)

func TestPasswordIsGeneratedOnceAndMarksAnUp(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/opt/signoz"] = 0o755
	g := noWait(Password(m, "/opt/signoz"))
	changed, err := step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false)
	if err != nil || !changed {
		t.Fatalf("first run = %v, %v; want a change", changed, err)
	}
	first := string(m.Files["/opt/signoz/root.env"])
	if !regexp.MustCompile(`^SIGNOZ_USER_ROOT_PASSWORD=[A-Za-z0-9]{32}-Aa1\n$`).MatchString(first) {
		t.Errorf("root.env = %q, want 32 letters and digits, then -Aa1", first)
	}
	if m.Modes["/opt/signoz/root.env"] != 0o600 {
		t.Errorf("root.env has mode %o, want 600", m.Modes["/opt/signoz/root.env"])
	}
	if _, ok := m.Files["/opt/signoz/recreate"]; !ok {
		t.Error("a new password did not mark the containers for recreation")
	}
	if changed, err = step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false); err != nil || changed {
		t.Errorf("second run = %v, %v; want none", changed, err)
	}
	if got := string(m.Files["/opt/signoz/root.env"]); got != first {
		t.Error("the second run replaced the password")
	}
}

func TestPasswordKeepsTheFileButFixesItsMode(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/opt/signoz/root.env"] = []byte("SIGNOZ_USER_ROOT_PASSWORD=mine\n")
	m.Modes["/opt/signoz/root.env"] = 0o644
	if _, err := step.Run(t.Context(), []step.Step{noWait(Password(m, "/opt/signoz"))}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/opt/signoz/root.env"]); got != "SIGNOZ_USER_ROOT_PASSWORD=mine\n" {
		t.Errorf("root.env = %q, want it kept", got)
	}
	if got := m.Modes["/opt/signoz/root.env"]; got != 0o600 {
		t.Errorf("root.env has mode %o, want 600", got)
	}
}

// noWait drops the wait on a step the test does not run.
func noWait(g host.Group) host.Group {
	g.After = ""
	return g
}
