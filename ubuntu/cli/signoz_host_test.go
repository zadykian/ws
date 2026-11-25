package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// The host runs SigNoz's steps as the guest does, over the host's values.
func TestHostSignozPasswordWritesOnce(t *testing.T) {
	p := rootProcess()
	m := p.Machine.(*effecttest.Machine)
	m.Dirs["/opt/signoz"] = 0o755
	if status, _, stderr := runWs(p, "", "ubuntu", "host", "signoz", "password"); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if first := string(m.Files["/opt/signoz/root.env"]); first == "" || m.Modes["/opt/signoz/root.env"] != 0o600 {
		t.Errorf("root.env = %q, mode %o; want a password, 600", first, m.Modes["/opt/signoz/root.env"])
	}
}

func TestHostSignozSyncCheckReadsTheHostsValues(t *testing.T) {
	if status, _, stderr := runWs(signozProcess(t, 90), "", "ubuntu", "host", "signoz", "sync", "--check"); status != 0 || stderr != "" {
		t.Errorf("status = %d, stderr = %q; want nothing to do", status, stderr)
	}
	status, _, stderr := runWs(signozProcess(t, 15), "", "ubuntu", "host", "signoz", "sync", "--check")
	if want := "ws ubuntu host signoz sync: signoz retention: set the retention of logs to 90 days"; status != 1 || !strings.Contains(stderr, want) {
		t.Errorf("status = %d, stderr = %q, want %q", status, stderr, want)
	}
}

func TestHostSignozSyncRefusesAGuestsKey(t *testing.T) {
	p := signozProcess(t, 90)
	m := p.Machine.(*effecttest.Machine)
	m.Files[hostVars] = append(m.Files[hostVars], "WS_ZED_NOFILE: 1\n"...)
	status, _, stderr := runWs(p, "", "ubuntu", "host", "signoz", "sync", "--check")
	if status != 1 || !strings.Contains(stderr, "WS_ZED_NOFILE") {
		t.Errorf("status = %d, stderr = %q; want the key refused", status, stderr)
	}
}
