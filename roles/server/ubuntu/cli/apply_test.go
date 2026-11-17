package cli

import (
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// serverProcess returns a process as root on the one system that ws server ubuntu
// apply takes, with ca-certificates and the base role's packages installed.
func serverProcess() (enginecli.Env, *effecttest.Machine) {
	p := fakeProcess()
	p.Euid = 0
	m := p.Machine.(*effecttest.Machine)
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Packages = effecttest.Packages{"ca-certificates": {Version: "1", Status: "ii"}}
	for _, name := range values.Default().Base.Packages {
		m.Packages[name] = effecttest.Package{Version: "1", Status: "ii"}
	}
	return p, m
}

// A role's steps follow the machine's values, so the check reads those first, and runs no step. A
// step's name holds a space, and --only keeps it whole.
func TestServerApplyRefusesAStepItsRoleLacks(t *testing.T) {
	p, m := serverProcess()
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply", "--only", "base/no such")
	want := `ws server ubuntu apply: no step "no such" in base; its steps here are ["Packages" "Locale" "Timezone" "Upgrades" "Swap" "Swappiness"]` + "\n" +
		"Run 'ws server ubuntu apply --help' for usage.\n"
	if status != 2 || stderr != want {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if want := []string{"read /etc/os-release", "read /etc/ws/values.yml"}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestServerApplyTakesAStepByName(t *testing.T) {
	p, _ := serverProcess()
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply", "--check", "--only", "base/Packages")
	if status != 0 || !strings.HasSuffix(stderr, "ws server ubuntu apply: would change: none\n") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
