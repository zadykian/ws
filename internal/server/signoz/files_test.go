package signoz

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
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

func TestFilesSetTheMarkBeforeTheyWriteAFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/opt"] = 0o755
	if _, err := step.Run(t.Context(), []step.Step{Files(m, "/opt/signoz")}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	mark, write := -1, -1
	for i, c := range m.Calls {
		switch {
		case mark < 0 && strings.HasPrefix(c, "write root /opt/signoz/recreate"):
			mark = i
		case write < 0 && strings.Contains(c, "/opt/signoz/compose.yaml") && !strings.HasPrefix(c, "read"):
			write = i
		}
	}
	if mark < 0 || write < 0 || mark > write {
		t.Errorf("mark at call %d, compose.yaml written at %d; want the mark first", mark, write)
	}
}

func TestFilesKeepTheMarkUntilAnUpClearsIt(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/opt"] = 0o755
	g := Files(m, "/opt/signoz")
	if _, err := step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	delete(m.Files, "/opt/signoz/recreate")
	delete(m.Root, "/opt/signoz/recreate")
	m.Files["/opt/signoz/compose.yaml"] = []byte("changed")
	if _, err := step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/opt/signoz/recreate"]; !ok {
		t.Error("a file that differs did not set the mark")
	}
}
