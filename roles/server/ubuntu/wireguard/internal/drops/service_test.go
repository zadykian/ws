package drops

import (
	"strings"
	"testing"
)

func TestDropsServiceRunsTheScriptOnTheInterfaces(t *testing.T) {
	got, err := service(MacInterfaces("enp5s0"))
	if err != nil {
		t.Fatal(err)
	}
	want := "ExecStart=/usr/local/lib/ws/wireguard/docker-user.sh wg0 enp5s0\n"
	if !strings.Contains(string(got), want) {
		t.Errorf("unit lacks %q:\n%s", want, got)
	}
	if !strings.HasPrefix(string(got), "# Written by github.com/zadykian/ws") || !strings.Contains(string(got), "WantedBy=docker.service") {
		t.Errorf("unit = %s", got)
	}
}
