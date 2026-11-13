package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func machineWithDaemons(names ...string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/Library/LaunchDaemons"] = 0o755
	for _, n := range names {
		m.Files["/Library/LaunchDaemons/com.github.zadykian.ws-tunnel."+n+".plist"] = []byte("x")
	}
	return m
}

func TestDaemonOfTakesTheFlagThePlistOrRefuses(t *testing.T) {
	for _, c := range []struct{ name, flag, want, fail string }{
		{"flag", "lab", "lab", ""},
		{"the one plist", "", "ws", ""},
		{"a bad name", "Lab", "", "--server Lab is not a server's name"},
	} {
		d, err := daemonOf(machineWithDaemons("ws"), c.flag)
		if (c.fail == "") != (err == nil) || d.Name != c.want || (err != nil && !strings.Contains(err.Error(), c.fail)) {
			t.Errorf("%s: daemonOf = %q, %v; want %q, %q", c.name, d.Name, err, c.want, c.fail)
		}
	}
	if _, err := daemonOf(machineWithDaemons(), ""); err == nil || !strings.Contains(err.Error(), "no tunnel daemon in /Library/LaunchDaemons") {
		t.Errorf("none: err = %v; want the refusal", err)
	}
	if _, err := daemonOf(machineWithDaemons("lab", "ws"), ""); err == nil || !strings.Contains(err.Error(), "several tunnel daemons, lab and ws: name one by --server") {
		t.Errorf("several: err = %v; want the refusal", err)
	}
}
