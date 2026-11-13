package host_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/host"
)

func TestSessionEnv(t *testing.T) {
	want := []string{"XDG_RUNTIME_DIR=/run/user/1000", "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus"}
	if got := host.SessionEnv(1000); !slices.Equal(got, want) {
		t.Errorf("SessionEnv = %q, want %q", got, want)
	}
}
