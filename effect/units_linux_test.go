//go:build linux

package effect

import (
	"context"
	"strings"
	"testing"

	"github.com/coreos/go-systemd/v22/dbus"
)

func TestUnitState(t *testing.T) {
	file := func(state string) []dbus.UnitFile {
		return []dbus.UnitFile{{Path: "/srv/systemd/system/a.service", Type: state}}
	}
	unit := func(active string) []dbus.UnitStatus {
		return []dbus.UnitStatus{{Name: "a.service", ActiveState: active}}
	}
	for _, tt := range []struct {
		files []dbus.UnitFile
		units []dbus.UnitStatus
		want  UnitState
	}{
		{nil, nil, UnitState{}},
		{file("enabled"), unit("active"), UnitState{Enabled: true, Active: true}},
		{file("static"), unit("inactive"), UnitState{Enabled: true}},
		{file("disabled"), unit("failed"), UnitState{Failed: true}},
		{file("masked"), unit("activating"), UnitState{}},
	} {
		if got := unitState(tt.files, tt.units, "a.service"); got != tt.want {
			t.Errorf("unitState(%v, %v) = %+v, want %+v", tt.files, tt.units, got, tt.want)
		}
	}
}

var _ Units = OSUnits{}

func TestOSUnitsRefuseABareName(t *testing.T) {
	ctx, u := context.Background(), OSUnits{}
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
		if err == nil || !strings.Contains(err.Error(), "suffix") {
			t.Errorf("%s on a bare name: %v", call, err)
		}
	}
}
