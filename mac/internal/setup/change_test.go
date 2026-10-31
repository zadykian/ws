package setup_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

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
