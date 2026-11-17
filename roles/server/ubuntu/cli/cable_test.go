package cli

import (
	"slices"
	"strings"
	"testing"
)

func TestCableCheckPassesWithoutAPort(t *testing.T) {
	p, m := valuesProcess("---\nWS_USER: root\n")
	if status, stdout, stderr := runWs(p, "", "server", "ubuntu", "cable", "check"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if want := []string{"read " + serverVars}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls %q, want %q alone", m.Calls, want)
	}
}

func TestCableCheckNamesAMissingPort(t *testing.T) {
	p, _ := valuesProcess("---\nWS_USER: root\nWS_CABLE_PORT: enp9s0\n")
	status, _, stderr := runWs(p, "", "server", "ubuntu", "cable", "check", "/etc/netplan/60-cable.yaml.ws-check")
	if status != 1 || !strings.Contains(stderr, "this machine has no interface enp9s0") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestCableCheckRefusesBadValues(t *testing.T) {
	p, _ := valuesProcess("---\nWS_CABLE_PORT: a&b\n")
	status, _, stderr := runWs(p, "", "server", "ubuntu", "cable", "check")
	if status != 1 || !strings.Contains(stderr, `WS_CABLE_PORT "a&b" is not an interface's name`) {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}
