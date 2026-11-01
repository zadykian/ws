//go:build linux

package effect

import (
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
