package setup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/step"
)

func TestAFileChangeNamesItsFileInAReport(t *testing.T) {
	tests := []struct {
		change step.Change
		want   string
	}{
		{setup.Write{Path: "/f", After: []byte("a\n")}, "would create /f"},
		{setup.Write{Path: "/f", Before: []byte("b\n"), After: []byte("a\n"), Exists: true}, "would write /f"},
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
		setup.MakeDir{Path: "/d", Mode: 0o700},
		setup.Write{Path: "/d/f", After: []byte("x\n"), Mode: 0o600},
		setup.SetMode{Path: "/d/f", Mode: 0o644},
		setup.Backup{Path: "/a", To: "/a~"},
	} {
		if err := setup.Apply(context.Background(), m, out, c); err != nil {
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

func TestAChangeSaysItsKind(t *testing.T) {
	tests := []struct {
		change step.Change
		want   step.ChangeKind
	}{
		{setup.Write{Path: "/f"}, step.Write},
		{setup.MakeDir{Path: "/d"}, step.Write},
		{setup.Install{Path: "/b"}, step.Install},
		{setup.Run{Name: "brew", Args: []string{"install"}}, step.Install},
		{setup.Run{Name: "open"}, step.OtherChange},
		{setup.Launchctl{}, step.Restart},
		{setup.Remark("a line"), step.OtherChange},
		{setup.KeyMake{}, step.OtherChange},
	}
	for _, tt := range tests {
		if got := step.KindOf(tt.change); got != tt.want {
			t.Errorf("%T: kind = %s, want %s", tt.change, got, tt.want)
		}
	}
}
