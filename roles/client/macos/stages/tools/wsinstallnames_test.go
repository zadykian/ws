package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

func TestWsInstallLinesNameTheServer(t *testing.T) {
	for held, want := range map[string]string{
		"v1.2.2": "would install ws v1.2.3 on lab",
		"v1.3.0": "lab holds v1.3.0",
	} {
		i, _ := newInstall("v1.2.3", releases, session.Status{Version: held})
		i.Server = "lab"
		p, err := i.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil {
			got = err.Error()
		}
		if !strings.Contains(got, want) {
			t.Errorf("held %s: %q; want %q", held, got, want)
		}
	}
	i, _ := newInstall("v1.2.3", releases, session.Status{})
	i.Server = "lab"
	if err := i.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "failed on lab") {
		t.Errorf("Apply = %v; want the stop to name lab", err)
	}
}
