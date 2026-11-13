package effecttest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
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

func TestUnitsRefuseABareName(t *testing.T) {
	ctx, u := context.Background(), &effecttest.Units{}
	_, state := u.State(ctx, "docker")
	if state == nil || !strings.Contains(state.Error(), "suffix") {
		t.Errorf("State on a bare name: %v", state)
	}
	for call, err := range map[string]error{
		"enable":       u.Enable(ctx, "docker.service", "docker"),
		"disable":      u.Disable(ctx, "docker"),
		"start":        u.Start(ctx, "docker"),
		"restart":      u.Restart(ctx, "docker"),
		"reload":       u.Reload(ctx, "docker"),
		"stop":         u.Stop(ctx, "docker"),
		"reset-failed": u.ResetFailed(ctx, "docker"),
	} {
		if err == nil || !strings.Contains(err.Error(), call) {
			t.Errorf("%s on a bare name: %v", call, err)
		}
	}
	if len(u.Calls) != 0 || len(u.States) != 0 {
		t.Errorf("a refused call changed the fake: %v %v", u.Calls, u.States)
	}
}
