package conf

import "testing"

const macPath = "/h/.config/ws/mac.conf"

func TestServerPathIsBesideMacConf(t *testing.T) {
	if got, want := ServerPath(macPath, "box"), "/h/.config/ws/servers/box/server.conf"; got != want {
		t.Errorf("ServerPath = %q, want %q", got, want)
	}
}
