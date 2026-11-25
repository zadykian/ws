package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/macos/values"
)

func TestSetupRefusesRoot(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	p.Euid = 0
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "ws macos setup: run it as your user, not root") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m)
}

func TestSetupTakesNoServer(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	status, stderr := runSetup(p, rel, "--server", "ws")
	if status != 1 || !strings.Contains(stderr, "setup takes no --server") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m)
}

func TestSetupRefusesASystemButAMacOnArm64(t *testing.T) {
	for _, got := range []string{"Linux aarch64", "Darwin x86_64"} {
		t.Run(got, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			m.Results["uname -sm"] = effecttest.Result{Out: []byte(got + "\n")}
			status, stderr := runSetup(p, rel)
			if status != 1 || !strings.Contains(stderr, "macOS on arm64 alone, not \""+got+"\"") {
				t.Errorf("status = %d, stderr = %q", status, stderr)
			}
			leftNothing(t, m)
		})
	}
}

func TestSetupRefusesABadValueBeforeAnyWrite(t *testing.T) {
	for _, args := range [][]string{nil, {"--plan"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			m.Files["/etc/ws/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "999"))
			status, stderr := runSetup(p, rel, args...)
			if status != 1 || !strings.Contains(stderr, "servers/ws/values.yml, line 1: WS_SLOT=999 is not a number") {
				t.Errorf("status = %d, stderr = %q", status, stderr)
			}
			leftNothing(t, m)
			if indexOf(m, "run xcode-select --install") >= 0 {
				t.Error("the command line tools were asked for before the values passed")
			}
		})
	}
}

func TestSetupRefusesAVersionThatNamesNoTag(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	p.Version = "devel"
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "names no release tag") || !strings.Contains(stderr, "--in-place") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m)
}

func TestSetupRefusesAMiseOfAnotherSHA256(t *testing.T) {
	rel := setupRelease("the mise that the release pins")
	p, m := setupProcess(rel)
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "SHA-256") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if _, ok := m.Files[setup.MisePath]; ok {
		t.Error("mise was written")
	}
}

func TestSetupGuardsTheTreeBeforeAnyWrite(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	m.Dirs["/opt/ws"] = 0o777
	for _, path := range []string{setup.MisePath, setup.MisercPath, setup.ConfigPath} {
		delete(m.Files, path)
	}
	delete(m.Results, setup.MisePath+" --version")
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "/opt/ws is not root's alone") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m)
}

func TestSetupInPlaceWantsATreeBeforeAnyWrite(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	status, stderr := runSetup(p, rel, "--in-place")
	if status != 1 || !strings.Contains(stderr, "holds no mise.toml") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	leftNothing(t, m)
}
