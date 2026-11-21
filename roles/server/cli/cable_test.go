package cli

import (
	"slices"
	"strings"
	"testing"
)

func TestCableCheckPassesWithoutAPort(t *testing.T) {
	p, m := valuesProcess("---\nWS_USER: root\n")
	if status, stdout, stderr := runWs(p, "", "server", "cable", "check"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if want := []string{"read " + serverVars}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls %q, want %q alone", m.Calls, want)
	}
}

func TestCableCheckNamesAMissingPort(t *testing.T) {
	p, _ := valuesProcess("---\nWS_USER: root\nWS_CABLE_PORT: enp9s0\n")
	status, _, stderr := runWs(p, "", "server", "cable", "check", "/etc/netplan/60-cable.yaml.ws-check")
	if status != 1 || !strings.Contains(stderr, "this machine has no interface enp9s0") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestCableCheckRefusesBadValues(t *testing.T) {
	p, _ := valuesProcess("---\nWS_CABLE_PORT: a&b\n")
	status, _, stderr := runWs(p, "", "server", "cable", "check")
	if status != 1 || !strings.Contains(stderr, `WS_CABLE_PORT "a&b" is not an interface's name`) {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestCableReloadWithoutSystemdIsDone(t *testing.T) {
	p, m := valuesProcess("---\n")
	for _, args := range [][]string{{"--check"}, nil} {
		status, stdout, stderr := runWs(p, "", append([]string{"server", "cable", "reload"}, args...)...)
		if status != 0 || stdout != "" || stderr != "" {
			t.Errorf("%q: status = %d, stdout %q, stderr %q", args, status, stdout, stderr)
		}
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "root ") }) {
		t.Errorf("calls %q, want no command", m.Calls)
	}
}

func TestCableReloadCheckFailsWhereDue(t *testing.T) {
	p, m := valuesProcess("---\n")
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/etc/netplan/60-cable.yaml"] = []byte("new\n")
	status, _, stderr := runWs(p, "", "server", "cable", "reload", "--check")
	if status != 1 || !strings.Contains(stderr, "to do: write networkd's files from netplan's") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
	if slices.Contains(m.Calls, "root netplan generate") {
		t.Errorf("--check ran netplan generate: %q", m.Calls)
	}
}
