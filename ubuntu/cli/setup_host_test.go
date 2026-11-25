package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
)

const hostBootstrapLine = "/usr/local/bin/mise -C /opt/ws/ubuntu/host bootstrap"

// hostProcess is setupProcess for the host: the machine's bootstrap runs in ubuntu/host.
func hostProcess(rel setup.MiseRelease) (enginecli.Env, *effecttest.Machine) {
	p, m := setupProcess(rel)
	delete(m.Results, bootstrapLine)
	m.Results[hostBootstrapLine] = effecttest.Result{}
	return p, m
}

// runHostSetup runs ws ubuntu host setup with args.
func runHostSetup(p enginecli.Env, rel setup.MiseRelease, args ...string) (status int, stderr string) {
	cfg := setupConfig{Mise: rel, Repo: treeRepo}
	group := enginecli.Group("ubuntu", "Ubuntu", enginecli.Group("host", "the host", newSetupCmd(p, cfg, hostSetup)))
	status, _, stderr = clitest.Run(p, []*cobra.Command{group}, "", append([]string{"ubuntu", "host", "setup"}, args...)...)
	return status, stderr
}

func TestHostSetupWritesTheHostsMiseAndRunsItsBootstrap(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := hostProcess(rel)
	if status, stderr := runHostSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if got := string(m.Files[setup.MisercPath]); got != string(setup.Miserc("host", setup.SelfContained)) {
		t.Errorf("miserc = %q", got)
	}
	if !ran(m, hostBootstrapLine) || ran(m, bootstrapLine) {
		t.Errorf("calls = %q", m.Calls)
	}
}

// A host has no secret: the clone that a guest's setup follows is none of its.
func TestHostSetupRunsNoTaskWhereAClonesStands(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := hostProcess(rel)
	m.Dirs[cloneDir] = 0o700
	if status, stderr := runHostSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if !ran(m, hostBootstrapLine) {
		t.Errorf("calls = %q", m.Calls)
	}
	for _, call := range m.Calls {
		if strings.Contains(call, " run secrets") {
			t.Errorf("a task ran: %s", call)
		}
	}
}

func TestHostSetupTakesNoSecretsFlag(t *testing.T) {
	rel := setupRelease(miseBody)
	p, _ := hostProcess(rel)
	if status, stderr := runHostSetup(p, rel, "--secrets"); status == 0 || !strings.Contains(stderr, "unknown flag: --secrets") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

const hostPlanLine = hostBootstrapLine + " plan --detailed-exitcode"

func TestHostSetupPlansTheHostsBootstrap(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := hostProcess(rel)
	standing(m, rel)
	m.Files[setup.MisercPath] = setup.Miserc("host", setup.SelfContained)
	m.Results[hostPlanLine] = effecttest.Result{}
	if status, stderr := runHostSetup(p, rel, "--plan"); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if !ran(m, hostPlanLine) {
		t.Errorf("calls = %q", m.Calls)
	}
}

func TestHostSetupChecksTheHostsValuesBeforeAnyWrite(t *testing.T) {
	for _, file := range []string{"---\nWS_BASE_SWAPPINESS: 400\n", "---\nWS_ZED_NOFILE: 1\n"} {
		rel := setupRelease(miseBody)
		p, m := hostProcess(rel)
		m.Files[hostVars] = []byte(file)
		status, stderr := runHostSetup(p, rel)
		if status != 1 || !strings.Contains(stderr, "edit "+hostVars+" by hand") {
			t.Errorf("%q: status = %d, stderr = %q", file, status, stderr)
		}
		leftNothing(t, m, false)
	}
}

func TestHostSetupRefusesAUser(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := hostProcess(rel)
	p.Euid = 1000
	if status, stderr := runHostSetup(p, rel); status != 1 || !strings.Contains(stderr, "ws ubuntu host setup: run it as root") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m, false)
}
