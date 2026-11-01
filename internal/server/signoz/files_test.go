package signoz

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestFilesCopyTheForgedFiles(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/opt"] = 0o755
	g := Files(m, "/opt/signoz")
	changed, err := step.Run(t.Context(), []step.Step{g}, out, false)
	if err != nil || !changed {
		t.Fatalf("first run = %v, %v; want a change and no error", changed, err)
	}
	for _, path := range []string{"/opt/signoz/compose.yaml", "/opt/signoz/ingester/opamp.yaml"} {
		if _, ok := m.Files[path]; !ok {
			t.Errorf("%s was not copied", path)
		}
	}
	if changed, err = step.Run(t.Context(), []step.Step{g}, out, false); err != nil || changed {
		t.Errorf("second run = %v, %v; want none", changed, err)
	}
}
