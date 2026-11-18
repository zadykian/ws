package cli

import (
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// serverProcess returns a process as root on the one system that ws server ubuntu
// apply takes, with ca-certificates installed.
func serverProcess() (enginecli.Env, *effecttest.Machine) {
	p := fakeProcess()
	p.Euid = 0
	m := p.Machine.(*effecttest.Machine)
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Packages = effecttest.Packages{"ca-certificates": {Version: "1", Status: "ii"}}
	return p, m
}

// No role is Go steps now, as each is mise's, so --only names none, and the check reads nothing.
// The tests of engine/core/role hold a step's name.
func TestServerApplyRefusesARoleWithNoneLeft(t *testing.T) {
	p, m := serverProcess()
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply", "--only", "devtools/no such")
	want := `ws server ubuntu apply: no role "devtools"; the roles are ` + "\n" +
		"Run 'ws server ubuntu apply --help' for usage.\n"
	if status != 2 || stderr != want {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls = %q, want none", m.Calls)
	}
}

// With no role, a run holds what every run starts with alone.
func TestServerApplyPreparesAloneWithNoRole(t *testing.T) {
	p, m := serverProcess()
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply", "--check")
	if status != 0 || !strings.HasSuffix(stderr, "ws server ubuntu apply: would change: none\n") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if want := []string{"read /etc/os-release", "read /etc/ws/values.yml"}; len(m.Calls) < 2 || !slices.Equal(m.Calls[:2], want) {
		t.Errorf("calls = %q, want them to start with %q", m.Calls, want)
	}
}
