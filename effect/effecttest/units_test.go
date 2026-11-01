package effecttest_test

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

func TestUnits(t *testing.T) {
	ctx := context.Background()
	u := &effecttest.Units{}
	_ = u.Enable(ctx, "a.service")
	_ = u.Start(ctx, "a.service")
	_ = u.DaemonReload(ctx)
	want := effect.UnitState{Enabled: true, Active: true}
	if got, _ := u.State(ctx, "a.service"); got != want {
		t.Errorf("State = %+v, want %+v", got, want)
	}
	_ = u.Stop(ctx, "a.service")
	if got, _ := u.State(ctx, "a.service"); got.Active {
		t.Errorf("State after Stop = %+v, want inactive", got)
	}
	if len(u.Calls) != 4 || u.Calls[2] != "daemon-reload" {
		t.Errorf("Calls = %v", u.Calls)
	}
}
