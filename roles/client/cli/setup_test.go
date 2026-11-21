package cli

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/internal/mactest"
	"github.com/zadykian/ws/roles/client/values"
	"github.com/zadykian/ws/roles/internal/setup"
)

const (
	miseBody  = "a mise that a test serves"
	treeRepo  = "https://example.org/ws"
	miseLine  = "/usr/local/bin/mise -C /opt/ws/roles/client bootstrap"
	planLine  = miseLine + " plan --detailed-exitcode"
	toolsLine = "xcode-select -p"
)

// miseExit is an error that carries a command's exit code, as an exec.ExitError does.
type miseExit int

func (e miseExit) Error() string { return "exit status" }
func (e miseExit) ExitCode() int { return int(e) }

// setupRelease returns a release of mise whose binary for macos-arm64 is miseBody.
func setupRelease(body string) setup.MiseRelease {
	return setup.MiseRelease{Version: "2026.1.1",
		Sums: map[string]string{"macos-arm64": fmt.Sprintf("%x", sha256.Sum256([]byte(body)))}}
}

// setupProcess returns the process of a user on a Mac on arm64 with the command line tools, good
// values and no tree yet, which serves miseBody. The process is of ws v1.2.3.
func setupProcess(rel setup.MiseRelease) (enginecli.Env, *effecttest.Machine) {
	p := withEnv(fakeProcess(), map[string]string{"WS_VALUES": "", "HOME": mactest.AgentHome})
	m := effecttest.NewMachine()
	p.Machine = m
	for _, dir := range []string{"/", "/usr", "/usr/local", "/usr/local/bin", "/usr/local/share", "/opt"} {
		m.Dirs[dir] = 0o755
	}
	writeConfs(m, values.Text("WS_GIT_SIGNING_PUB_KEY", "~/g.pub", "WS_IDENTITY_PUB_KEY", "~/i.pub"), values.Text("WS_SLOT", "0"))
	m.Results["uname -sm"] = effecttest.Result{Out: []byte("Darwin arm64\n")}
	m.Results[toolsLine] = effecttest.Result{Out: []byte("/Library/Developer/CommandLineTools\n")}
	m.Results["git --version"] = effecttest.Result{Out: []byte("git version 2.50.0\n")}
	for _, line := range []string{
		"git -c init.defaultBranch=ws init -q /opt/ws",
		"git -C /opt/ws fetch -q --depth 1 --force " + treeRepo + " refs/tags/v1.2.3:refs/tags/v1.2.3",
		"git -C /opt/ws -c advice.detachedHead=false checkout -q --force --detach refs/tags/v1.2.3",
		"git -C /opt/ws clean -q -ffdx",
		miseLine,
	} {
		m.Results[line] = effecttest.Result{}
	}
	m.Pages[rel.URL("macos-arm64")] = []byte(miseBody)
	return p, m
}

// standing makes the tree, mise, mise's files and the plugin stand as a run leaves them, with the
// tree's checkout at the tag and clean.
func standing(m *effecttest.Machine, rel setup.MiseRelease) {
	m.Files[setup.MisePath] = []byte(miseBody)
	m.Results[setup.MisePath+" --version"] = effecttest.Result{Out: []byte(rel.Version + " macos-arm64 (2026-01-01)\n")}
	m.Files[setup.MisercPath] = setup.Miserc("mac")
	m.Files[setup.ConfigPath] = setup.Config()
	for _, dir := range []string{"/etc/mise", "/opt/ws", "/opt/ws/.git", "/usr/local/share/mise", macPluginDir} {
		m.Dirs[dir] = 0o755
	}
	m.Files["/opt/ws/mise.toml"] = nil
	m.Links[macPluginDir+"/ws"] = "/opt/ws/plugins/ws"
	read := "git -c safe.directory=/opt/ws -C /opt/ws "
	for _, line := range []string{"HEAD^{commit}", "refs/tags/v1.2.3^{commit}"} {
		m.Results[read+"rev-parse --verify -q "+line] = effecttest.Result{Out: []byte("abc123\n")}
	}
	m.Results[read+"--no-optional-locks status --porcelain --ignored"] = effecttest.Result{}
}

func runSetup(p enginecli.Env, rel setup.MiseRelease, args ...string) (status int, stderr string) {
	return runSetupWith(p, setupConfig{Mise: rel, Repo: treeRepo, Poll: time.Millisecond}, args...)
}

func runSetupWith(p enginecli.Env, cfg setupConfig, args ...string) (status int, stderr string) {
	var server string
	group := enginecli.Group("client", "the Mac", newSetupCmd(p, &server, cfg))
	group.PersistentFlags().StringVar(&server, "server", "", "")
	status, _, stderr = clitest.Run(p, []*cobra.Command{group}, "", append([]string{"client", "setup"}, args...)...)
	return status, stderr
}

func indexOf(m *effecttest.Machine, call string) int {
	return slices.IndexFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, call) })
}

// noWrites fails the test where a call wrote, or asked sudo for anything.
func noWrites(t *testing.T, m *effecttest.Machine) {
	for _, call := range m.Calls {
		for _, write := range []string{"atomic ", "mkdir ", "symlink ", "chown ", "chmod ", "download ", "root "} {
			if strings.HasPrefix(call, write) {
				t.Errorf("a write: %s", call)
			}
		}
	}
}

// leftNothing is noWrites, and fails the test where the machine holds a file that setup writes, or
// mise's bootstrap ran.
func leftNothing(t *testing.T, m *effecttest.Machine) {
	noWrites(t, m)
	for _, path := range []string{setup.MisePath, setup.MisercPath, setup.ConfigPath} {
		if _, ok := m.Files[path]; ok {
			t.Errorf("%s was written", path)
		}
	}
	if indexOf(m, "run "+miseLine) >= 0 {
		t.Error("the bootstrap ran")
	}
}

func TestSetupInstallsMiseWritesItsFilesAndRunsTheBootstrapLast(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if got := string(m.Files[setup.MisePath]); got != miseBody || m.Modes[setup.MisePath] != 0o755 {
		t.Errorf("mise = %q at mode %o", got, m.Modes[setup.MisePath])
	}
	if got := string(m.Files[setup.MisercPath]); got != string(setup.Miserc("mac")) {
		t.Errorf("miserc = %q", got)
	}
	order := []string{"download " + rel.URL("macos-arm64"), "atomic /etc/mise/config.toml", "root git -c init.defaultBranch=ws init",
		"root git -C /opt/ws fetch", "root git -C /opt/ws -c advice.detachedHead=false checkout", "root git -C /opt/ws clean",
		"symlink /opt/ws/plugins/ws " + macPluginDir + "/ws", "run " + miseLine}
	last := -1
	for _, call := range order {
		at := indexOf(m, call)
		if at <= last {
			t.Errorf("%q is at %d, after %d: %q", call, at, last, m.Calls)
		}
		last = at
	}
}

func TestSetupMakesTheDirectoriesAMacLacks(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	for _, dir := range []string{"/usr/local", "/usr/local/bin", "/usr/local/share", "/opt"} {
		delete(m.Dirs, dir)
	}
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	for _, dir := range []string{"/usr/local", "/usr/local/bin", "/usr/local/share", "/opt", "/usr/local/share/mise", macPluginDir} {
		if m.Dirs[dir] != 0o755 {
			t.Errorf("%s has mode %o, want 755", dir, m.Dirs[dir])
		}
	}
}

func TestSetupAgainChangesNothingAndAsksNoSudo(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	if status, stderr := runSetup(p, rel); status != 0 || strings.Contains(stderr, "write") {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	noWrites(t, m)
	if indexOf(m, "run "+miseLine) < 0 {
		t.Errorf("no bootstrap: %q", m.Calls)
	}
}

func TestSetupInPlaceRunsTheTreeThatStandsOfAnyVersion(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	p.Version = "devel"
	if status, stderr := runSetup(p, rel, "--in-place"); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	noWrites(t, m)
}

func TestSetupEntersBeforeItsSteps(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	entered := false
	cfg := setupConfig{Mise: rel, Repo: treeRepo, Enter: func() error {
		entered = indexOf(m, "stat") < 0 && indexOf(m, "run git") < 0
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
	m.Results[miseLine] = effecttest.Result{Err: errors.New("a unit failed")}
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
	leftNothing(t, m)
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
			if indexOf(m, "run "+planLine) < 0 || slices.Contains(m.Calls, "run "+miseLine) {
				t.Errorf("calls = %q", m.Calls)
			}
		})
	}
}
