package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
)

const (
	bootstrapLine = "/usr/local/bin/mise -C /opt/ws/ubuntu/guest bootstrap"
	planLine      = bootstrapLine + " plan --detailed-exitcode"
)

// miseExit is an error that carries a command's exit code, as an exec.ExitError does.
type miseExit int

func (e miseExit) Error() string { return "exit status" }
func (e miseExit) ExitCode() int { return int(e) }

// standing makes the tree, mise, mise's files and the plugin stand as a run leaves them, with the
// tree's checkout at the tag and clean.
func standing(m *effecttest.Machine, rel setup.MiseRelease) {
	m.Files[setup.MisePath] = []byte(miseBody)
	m.Results[setup.MisePath+" --version"] = effecttest.Result{Out: []byte(rel.Version + " linux-x64 (2026-01-01)\n")}
	m.Files[setup.MisercPath] = setup.Miserc("guest")
	m.Files[setup.ConfigPath] = setup.Config()
	m.Dirs["/etc/mise"] = 0o755
	for _, dir := range []string{"/opt/ws", "/opt/ws/.git", "/usr/share/mise", setup.PluginDir} {
		m.Dirs[dir] = 0o755
	}
	m.Files["/opt/ws/mise.toml"] = nil
	m.Links[setup.PluginDir+"/ws"] = "/opt/ws/plugins/ws"
	for _, line := range []string{"HEAD^{commit}", "refs/tags/v1.2.3^{commit}"} {
		m.Results["git -C /opt/ws rev-parse --verify -q "+line] = effecttest.Result{Out: []byte("abc123\n")}
	}
	m.Results["git -C /opt/ws --no-optional-locks status --porcelain --ignored"] = effecttest.Result{}
}

func indexOf(m *effecttest.Machine, call string) int {
	return slices.IndexFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, call) })
}

func TestSetupChecksTheTagOutAndRunsTheBootstrapAfterTheSteps(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	order := []string{"atomic /etc/mise/config.toml", "run git -c init.defaultBranch=ws init", "run git -C /opt/ws fetch",
		"run git -C /opt/ws -c advice.detachedHead=false checkout", "run git -C /opt/ws clean",
		"symlink /opt/ws/plugins/ws " + setup.PluginDir + "/ws", "run " + bootstrapLine}
	last := -1
	for _, call := range order {
		at := indexOf(m, call)
		if at <= last {
			t.Errorf("%q is at %d, after %d: %q", call, at, last, m.Calls)
		}
		last = at
	}
}

func TestSetupRunsTheBootstrapAfterEachRunAndWritesNothingTwice(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	if status, stderr := runSetup(p, rel); status != 0 || strings.Contains(stderr, "write") {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	for _, call := range m.Calls {
		for _, write := range []string{"atomic ", "mkdir ", "symlink ", "download ", "run git -c ", "run git -C /opt/ws fetch",
			"run git -C /opt/ws clean"} {
			if strings.HasPrefix(call, write) {
				t.Errorf("the second run wrote: %s", call)
			}
		}
	}
	if indexOf(m, "run "+bootstrapLine) < 0 {
		t.Errorf("no bootstrap: %q", m.Calls)
	}
}

func TestSetupRefusesAVersionThatNamesNoTag(t *testing.T) {
	for _, version := range []string{"devel", "v0.13.1-0.20261101120000-abcdef123456+dirty", "v1.2.3+dirty"} {
		t.Run(version, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			p.Version = version
			status, stderr := runSetup(p, rel)
			if status != 1 || !strings.Contains(stderr, "names no release tag") || !strings.Contains(stderr, "--in-place") {
				t.Errorf("status = %d, stderr = %q", status, stderr)
			}
			leftNothing(t, m, false)
		})
	}
}

func TestSetupInPlaceRunsTheTreeThatStandsOfAnyVersion(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	p.Version = "devel"
	m.Results[installedWs+" version"] = effecttest.Result{Out: []byte("devel\n")}
	if status, stderr := runSetup(p, rel, "--in-place"); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "run git -C /opt/ws fetch") || strings.HasPrefix(call, "run git -C /opt/ws clean") {
			t.Errorf("the tree was touched: %s", call)
		}
	}
	if indexOf(m, "run "+bootstrapLine) < 0 {
		t.Errorf("no bootstrap: %q", m.Calls)
	}
}

func TestSetupInPlaceWantsATreeBeforeAnyWrite(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	status, stderr := runSetup(p, rel, "--in-place")
	if status != 1 || !strings.Contains(stderr, "holds no mise.toml") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m, false)
}

func TestSetupGuardsTheTreeBeforeAnyWrite(t *testing.T) {
	for name, change := range map[string]func(m *effecttest.Machine){
		"a file of another owner": func(m *effecttest.Machine) { m.Owners["/opt/ws/mise.toml"] = [2]int{1000, 1000} },
		"a tree others write":     func(m *effecttest.Machine) { m.Dirs["/opt/ws"] = 0o777 },
		"a tree that is a link": func(m *effecttest.Machine) {
			delete(m.Dirs, "/opt/ws")
			m.Links["/opt/ws"] = "/srv/ws"
		},
		"an /opt a group writes": func(m *effecttest.Machine) { m.Dirs["/opt"] = 0o775 },
	} {
		t.Run(name, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			standing(m, rel)
			delete(m.Files, setup.ConfigPath)
			delete(m.Files, setup.MisercPath)
			delete(m.Files, setup.MisePath)
			delete(m.Results, setup.MisePath+" --version")
			change(m)
			status, stderr := runSetup(p, rel)
			if status != 1 || !strings.Contains(stderr, "root's") {
				t.Errorf("status = %d, stderr = %q", status, stderr)
			}
			leftNothing(t, m, false)
		})
	}
}

func TestSetupEntersBeforeItsSteps(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	entered := false
	cfg := setupConfig{Mise: rel, Repo: treeRepo, Enter: func() error {
		entered = indexOf(m, "stat") < 0 && indexOf(m, "run ") < 0
		return nil
	}}
	runSetupWith(p, cfg)
	if !entered {
		t.Error("Enter did not run first")
	}
}

func TestSetupStopsWhereTheBootstrapFails(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	m.Results[bootstrapLine] = effecttest.Result{Err: errors.New("a unit failed")}
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "a unit failed") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestSetupPlanSaysWhatItWouldDoAndChangesNothing(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	status, stderr := runSetup(p, rel, "--plan")
	if status != 2 || !strings.Contains(stderr, "would check out v1.2.3 of "+treeRepo+" in /opt/ws") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m, false)
}

func TestSetupPlanAsksMisesPlanWhereTheStepsHaveNothingToDo(t *testing.T) {
	for name, tc := range map[string]struct {
		result effecttest.Result
		status int
	}{
		"nothing":     {effecttest.Result{}, 0},
		"a change":    {effecttest.Result{Err: miseExit(2)}, 2},
		"a failure":   {effecttest.Result{Err: miseExit(1)}, 1},
		"a lost mise": {effecttest.Result{Err: errors.New("no mise")}, 1},
	} {
		t.Run(name, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			standing(m, rel)
			m.Results[planLine] = tc.result
			status, _ := runSetup(p, rel, "--plan")
			if status != tc.status {
				t.Errorf("status = %d, want %d", status, tc.status)
			}
			if indexOf(m, "run "+bootstrapLine+" plan") < 0 || slices.Contains(m.Calls, "run "+bootstrapLine) {
				t.Errorf("calls = %q", m.Calls)
			}
		})
	}
}

func TestSetupPlanLeavesMisesPlanOutWhereAStepIsDue(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	m.Results["git -C /opt/ws rev-parse --verify -q HEAD^{commit}"] = effecttest.Result{Out: []byte("old\n")}
	status, stderr := runSetup(p, rel, "--plan")
	if status != 2 || !strings.Contains(stderr, "would check out v1.2.3") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if indexOf(m, "run "+bootstrapLine) >= 0 {
		t.Errorf("mise's plan ran: %q", m.Calls)
	}
}
