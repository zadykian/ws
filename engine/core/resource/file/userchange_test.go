package file_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

func TestAFileChangeNamesItsFileInAReport(t *testing.T) {
	tests := []struct {
		change step.Change
		want   string
	}{
		{file.UserWrite{Path: "/f", After: []byte("a\n")}, "would create /f"},
		{file.UserWrite{Path: "/f", Before: []byte("b\n"), After: []byte("a\n"), Exists: true}, "would write /f"},
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

// ws replaces its own files whole, so what one held stands in the excerpt.
func TestAnExcerptKeepsTheDiffOfAFileWsReplaces(t *testing.T) {
	w := file.UserWrite{Path: "/f", Before: []byte("old\n"), After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("excerpt = %q, want the diff %q", got, w.String())
	}
	w = file.UserWrite{Path: "/f", After: []byte("new\n"), Exists: true}
	if got := step.ExcerptOf(w); got != w.String() {
		t.Errorf("empty file: excerpt = %q, want the diff %q", got, w.String())
	}
}

func TestAUserChangeSaysItsKind(t *testing.T) {
	for _, c := range []step.Change{file.UserWrite{Path: "/f"}, file.UserMkdir{Path: "/d"}, file.UserChmod{Path: "/f"}} {
		if got := step.KindOf(c); got != step.Write {
			t.Errorf("%T: kind = %s, want %s", c, got, step.Write)
		}
	}
}

func TestAUserChangeAppliesItself(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["cp -p /a /a~"] = effecttest.Result{}
	for _, c := range []interface{ Apply(context.Context) error }{
		file.UserMkdir{M: m, Path: "/d", Mode: 0o700},
		file.UserWrite{M: m, Out: out, Path: "/d/f", After: []byte("x\n"), Mode: 0o600},
		file.UserChmod{M: m, Path: "/d/f", Mode: 0o644},
		file.UserBackup{M: m, Out: out, Path: "/a", To: "/a~"},
	} {
		if err := c.Apply(context.Background()); err != nil {
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

func TestAUserChangeReturnsItsCauseBare(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["cp -p /a /a~"] = effecttest.Result{Err: errors.New("disk full")}
	err := file.UserBackup{M: m, Out: &effecttest.Out{}, Path: "/a", To: "/a~"}.Apply(context.Background())
	if err == nil || err.Error() != "disk full" {
		t.Errorf("Apply = %v, want the bare cause", err)
	}
}
