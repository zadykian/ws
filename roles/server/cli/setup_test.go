package cli

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
)

const miseBody = "a mise that a test serves"

// setupRelease returns a release of mise whose binary for linux-x64 is miseBody.
func setupRelease(body string) setup.MiseRelease {
	return setup.MiseRelease{Version: "2026.1.1",
		Sums: map[string]string{"linux-x64": fmt.Sprintf("%x", sha256.Sum256([]byte(body)))}}
}

// treeRepo is where the tests' setup fetches the tree from.
const treeRepo = "https://example.org/ws"

// setupProcess returns a root process on an Ubuntu 26.04 machine on amd64 that serves miseBody and
// has no tree yet. The process is of ws v1.2.3, as is the installed ws.
func setupProcess(rel setup.MiseRelease) (enginecli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	for _, dir := range aboveTree {
		m.Dirs[dir] = 0o755
	}
	m.Results["git --version"] = effecttest.Result{Out: []byte("git version 2.50.0\n")}
	for _, line := range []string{
		"git -c init.defaultBranch=ws init -q /opt/ws",
		"git -C /opt/ws fetch -q --depth 1 --force " + treeRepo + " refs/tags/v1.2.3:refs/tags/v1.2.3",
		"git -C /opt/ws -c advice.detachedHead=false checkout -q --force --detach refs/tags/v1.2.3",
		"git -C /opt/ws clean -q -ffdx",
		bootstrapLine,
	} {
		m.Results[line] = effecttest.Result{}
	}
	m.Files[osRelease] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[installedWs+" version"] = effecttest.Result{Out: []byte(p.Version + "\n")}
	m.Pages[rel.URL("linux-x64")] = []byte(miseBody)
	return p, m
}

// runSetup runs ws server setup with args, as a run that fetches the tree from treeRepo.
func runSetup(p enginecli.Env, rel setup.MiseRelease, args ...string) (status int, stderr string) {
	return runSetupWith(p, setupConfig{Mise: rel, Repo: treeRepo}, args...)
}

func runSetupWith(p enginecli.Env, cfg setupConfig, args ...string) (status int, stderr string) {
	group := enginecli.Group("server", "the server", newSetupCmd(p, cfg))
	status, _, stderr = clitest.Run(p, []*cobra.Command{group}, "", append([]string{"server", "setup"}, args...)...)
	return status, stderr
}

// leftNothing fails the test where the machine holds mise or a file of /etc/mise, or a call wrote:
// a file, a directory, a link or the tree. A refusal before any write fetched nothing either, and
// tried says that a download was tried.
func leftNothing(t *testing.T, m *effecttest.Machine, tried bool) {
	for _, path := range []string{setup.MisePath, setup.MisercPath, setup.ConfigPath} {
		if _, ok := m.Files[path]; ok {
			t.Errorf("%s was written", path)
		}
	}
	for _, call := range m.Calls {
		for _, write := range []string{"atomic ", "mkdir ", "symlink ", "chown ", "run git -c ", "run git -C /opt/ws fetch",
			"run git -C /opt/ws clean", "run " + bootstrapLine} {
			if strings.HasPrefix(call, write) {
				t.Errorf("a write: %s", call)
			}
		}
		if strings.HasPrefix(call, "download ") && !tried {
			t.Errorf("a write: %s", call)
		}
	}
}

func TestSetupRefusesAUser(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	p.Euid = 1000
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "ws server setup: run it as root") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m, false)
}

func TestSetupRefusesABadValueBeforeAnyWrite(t *testing.T) {
	for _, args := range [][]string{nil, {"--plan"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			m.Files[serverVars] = []byte("---\nWS_BASE_SWAPPINESS: 400\n")
			status, stderr := runSetup(p, rel, args...)
			if status != 1 || !strings.Contains(stderr, "WS_BASE_SWAPPINESS is 400") || !strings.Contains(stderr, "edit "+serverVars+" by hand") {
				t.Errorf("status = %d, stderr = %q", status, stderr)
			}
			leftNothing(t, m, false)
		})
	}
}

func TestSetupRefusesASystemButUbuntu2604(t *testing.T) {
	for name, text := range map[string]string{
		"24.04":  "ID=ubuntu\nVERSION_ID=\"24.04\"\n",
		"debian": "ID=debian\nVERSION_ID=\"13\"\n",
		"none":   "",
	} {
		t.Run(name, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			if text == "" {
				delete(m.Files, osRelease)
			} else {
				m.Files[osRelease] = []byte(text)
			}
			status, stderr := runSetup(p, rel)
			if status != 1 || !strings.Contains(stderr, "Ubuntu 26.04 alone") {
				t.Errorf("status = %d, stderr = %q", status, stderr)
			}
			leftNothing(t, m, false)
		})
	}
}

func TestSetupRefusesAnArchitectureButAmd64AndArm64(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("i386\n")}
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "x86_64 and aarch64, not on i386") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m, false)
}

func TestSetupRefusesAMiseOfAnotherSHA256(t *testing.T) {
	rel := setupRelease("the mise that the release pins")
	p, m := setupProcess(rel)
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "SHA-256") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m, true)
}

func TestSetupInstallsMiseAndWritesItsFiles(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	status, stderr := runSetup(p, rel)
	if status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if got := string(m.Files[setup.MisePath]); got != miseBody || m.Modes[setup.MisePath] != 0o755 {
		t.Errorf("mise = %q at mode %o", got, m.Modes[setup.MisePath])
	}
	if got := string(m.Files[setup.MisercPath]); got != string(setup.Miserc("server")) {
		t.Errorf("miserc = %q", got)
	}
	if got := string(m.Files[setup.ConfigPath]); got != string(setup.Config()) {
		t.Errorf("config = %q", got)
	}
}

func TestSetupAgainWritesNothing(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("first run: status = %d, stderr = %q", status, stderr)
	}
	// The fake's git writes leave no checkout, so the tree stands as a first run leaves it.
	standing(m, rel)
	m.Calls = nil
	if status, stderr := runSetup(p, rel); status != 0 || strings.Contains(stderr, "write") {
		t.Fatalf("second run: status = %d, stderr = %q", status, stderr)
	}
	// The bootstrap runs on each run by design, and no check counts it.
	for _, call := range m.Calls {
		for _, write := range []string{"download ", "atomic ", "run git -c ", "run git -C /opt/ws fetch",
			"run git -C /opt/ws checkout", "run git -C /opt/ws -c advice.detachedHead=false checkout",
			"run git -C /opt/ws clean", "run git -C /opt/ws remote"} {
			if strings.HasPrefix(call, write) {
				t.Errorf("the second run wrote: %s", call)
			}
		}
	}
}

func TestSetupInstallsMiseAgainWhereTheVersionDrifts(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	m.Files[setup.MisePath] = []byte("an older mise")
	m.Results[setup.MisePath+" --version"] = effecttest.Result{Out: []byte("2025.1.1 linux-x64 (2025-01-01)\n")}
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if got := string(m.Files[setup.MisePath]); got != miseBody {
		t.Errorf("mise = %q", got)
	}
}
