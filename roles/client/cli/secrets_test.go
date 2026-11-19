package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestSecretsInitRefusesRoot(t *testing.T) {
	p := fakeProcess()
	p.Euid = 0
	status, _, stderr := runWs(p, "client", "macos", "secrets", "init")
	if status != 1 || !strings.Contains(stderr, "run it as yourself, not root") {
		t.Errorf("init as root: status = %d, stderr = %q", status, stderr)
	}
	if calls := p.Machine.(*effecttest.Machine).Calls; len(calls) != 0 {
		t.Errorf("init as root ran %q", calls)
	}
}

func TestSecretsInitTakesNoServer(t *testing.T) {
	status, _, stderr := runWs(fakeProcess(), "client", "macos", "secrets", "init", "--server", "ws")
	if status != 1 || !strings.Contains(stderr, "init takes no --server") {
		t.Errorf("init --server ws: status = %d, stderr = %q", status, stderr)
	}
}

func TestSecretsInitTakesNoArgument(t *testing.T) {
	status, _, stderr := runWs(fakeProcess(), "client", "macos", "secrets", "init", "ws")
	if status != 2 || !strings.Contains(stderr, `unknown command "ws"`) {
		t.Errorf("init ws: status = %d, stderr = %q", status, stderr)
	}
}

func TestSecretsInitReadsTheTree(t *testing.T) {
	status, _, stderr := runWs(fakeProcess(), "client", "macos", "secrets", "init")
	if status != 1 || !strings.Contains(stderr, "/opt/ws/mise.toml") {
		t.Errorf("init without a tree: status = %d, stderr = %q", status, stderr)
	}
}
