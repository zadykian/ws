package change_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

func TestAFileChangeNamesItsFileInAReport(t *testing.T) {
	tests := []struct {
		change step.Change
		want   string
	}{
		{change.Write{Path: "/f", After: []byte("a\n")}, "would create /f"},
		{change.Write{Path: "/f", Before: []byte("b\n"), After: []byte("a\n"), Exists: true}, "would write /f"},
	}
	for _, tt := range tests {
		got, _, _ := strings.Cut(step.PendingOf(tt.change).Text, "\n")
		if got != tt.want {
			t.Errorf("%T: first line = %q, want %q", tt.change, got, tt.want)
		}
		if !strings.Contains(step.PendingOf(tt.change).Text, tt.change.String()) {
			t.Errorf("%T: the report lost the diff", tt.change)
		}
	}
}

func TestApply(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["cp -p /a /a~"] = effecttest.Result{}
	for _, c := range []interface{ String() string }{
		change.MakeDir{Path: "/d", Mode: 0o700},
		change.Write{Path: "/d/f", After: []byte("x\n"), Mode: 0o600},
		change.SetMode{Path: "/d/f", Mode: 0o644},
		change.Backup{Path: "/a", To: "/a~"},
		change.Remark("a line"),
	} {
		if err := change.Apply(context.Background(), m, out, c); err != nil {
			t.Fatalf("Apply(%v): %v", c, err)
		}
	}
	if info, err := m.Stat("/d/f"); err != nil || info.Mode != 0o644 {
		t.Errorf("Stat(/d/f) = %+v, %v; want mode 644", info, err)
	}
	if got := m.Dirs["/d"]; got != 0o700 {
		t.Errorf("mode of /d = %o, want 700", got)
	}
	want := []string{"say: wrote /d/f", "say: kept the old /a as /a~"}
	if len(out.Lines) != 2 || out.Lines[0] != want[0] || out.Lines[1] != want[1] {
		t.Errorf("said %q, want %q", out.Lines, want)
	}
}

func TestApplySaysAHostChangeByItsHeadline(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	cs, err := file.File{M: m, Path: "/e/f", Content: []byte("x\n"), Mode: 0o644}.Plan(context.Background())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v; want one change", cs, err)
	}
	if err := change.Apply(context.Background(), m, out, cs[0]); err != nil {
		t.Fatal(err)
	}
	if string(m.Files["/e/f"]) != "x\n" || len(out.Lines) != 1 || out.Lines[0] != "say: create /e/f" {
		t.Errorf("/e/f holds %q, and Apply said %q", m.Files["/e/f"], out.Lines)
	}
}

func TestAChangeSaysItsKind(t *testing.T) {
	tests := []struct {
		change step.Change
		want   step.ChangeKind
	}{
		{change.Write{Path: "/f"}, step.Write},
		{change.MakeDir{Path: "/d"}, step.Write},
		{change.Run{Name: "brew", Args: []string{"install"}}, step.Install},
		{change.Run{Name: "open"}, step.OtherChange},
		{change.Remark("a line"), step.OtherChange},
	}
	for _, tt := range tests {
		if got := step.KindOf(tt.change); got != tt.want {
			t.Errorf("%T: kind = %s, want %s", tt.change, got, tt.want)
		}
	}
}
