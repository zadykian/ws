package setup_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

func TestRootDirAndFixApply(t *testing.T) {
	m := effecttest.NewMachine()
	for _, line := range []string{"mkdir -m 755 /etc/x", "chown 0:0 /etc/x", "chmod 700 /etc/x"} {
		m.Results[line] = effecttest.Result{}
	}
	for _, c := range []step.Change{setup.RootDir{Path: "/etc/x", Mode: 0o755}, setup.RootFix{Path: "/etc/x", Mode: 0o700}} {
		if err := setup.Apply(context.Background(), m, &effecttest.Out{}, c); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(m.Calls, ";"); got != "root mkdir -m 755 /etc/x;root chown 0:0 /etc/x;root chown 0:0 /etc/x;root chmod 700 /etc/x" {
		t.Errorf("calls = %q", got)
	}
}

func TestRootChangeLines(t *testing.T) {
	if got, want := (setup.RootDir{Path: "/d", Mode: 0o700}).String(), "would make /d, root's, with mode 700"; got != want {
		t.Errorf("RootDir = %q; want %q", got, want)
	}
	f := setup.RootFix{Path: "/d", Mode: 0o755, Was: "501:20 700"}
	if got, want := f.String(), "would make /d root's, with mode 755, rather than 501:20 700"; got != want {
		t.Errorf("RootFix = %q; want %q", got, want)
	}
}
