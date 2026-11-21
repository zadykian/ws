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
	"github.com/zadykian/ws/roles/internal/setup"
)

const miseBody = "a mise that a test serves"

// setupRelease returns a release of mise whose binary for linux-x64 is miseBody.
func setupRelease(body string) setup.MiseRelease {
	return setup.MiseRelease{Version: "2026.1.1",
		Sums: map[string]string{"linux-x64": fmt.Sprintf("%x", sha256.Sum256([]byte(body)))}}
}

// setupProcess returns a root process on an Ubuntu 26.04 machine on amd64 that serves miseBody.
func setupProcess(rel setup.MiseRelease) (enginecli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	m.Files[osRelease] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Pages[rel.URL("linux-x64")] = []byte(miseBody)
	return p, m
}

func runSetup(p enginecli.Env, rel setup.MiseRelease) (status int, stderr string) {
	group := enginecli.Group("server", "the server", newSetupCmd(p, rel))
	status, _, stderr = clitest.Run(p, []*cobra.Command{group}, "", "server", "setup")
	return status, stderr
}

// leftNothing fails the test where the machine holds mise or a file of /etc/mise, or a call wrote.
// A refusal before any write fetched nothing either, and tried says that a download was tried.
func leftNothing(t *testing.T, m *effecttest.Machine, tried bool) {
	for _, path := range []string{setup.MisePath, setup.MisercPath, setup.ConfigPath} {
		if _, ok := m.Files[path]; ok {
			t.Errorf("%s was written", path)
		}
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "atomic ") || strings.HasPrefix(call, "download ") && !tried {
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
	m.Results[setup.MisePath+" --version"] = effecttest.Result{Out: []byte(rel.Version + " linux-x64 (2026-01-01)\n")}
	m.Calls = nil
	if status, stderr := runSetup(p, rel); status != 0 || strings.Contains(stderr, "write") {
		t.Fatalf("second run: status = %d, stderr = %q", status, stderr)
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "download ") || strings.HasPrefix(call, "atomic ") {
			t.Errorf("the second run wrote: %s", call)
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
