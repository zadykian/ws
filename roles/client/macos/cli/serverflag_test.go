package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

func TestServerFlagNamesTheServerOfApplyAndState(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, "", "WS_USER=ada\n")
	m.Files[conf.ServerPath("/h/.config/ws/mac.conf", "box")] = []byte("WS_USER=bob\n")
	m.Dirs["/h/.config/ws/servers/box"] = 0o755
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = m
	want := "several servers in /h/.config/ws/servers, box and ws: name one by --server"
	if status, _, stderr := runWs(p, "", "client", "macos", "state"); status != 1 || !strings.Contains(stderr, want) {
		t.Errorf("no flag: status = %d, stderr = %q; want %q", status, stderr, want)
	}
	if status, stdout, stderr := runWs(p, "", "client", "macos", "state", "--server", "box", "--json"); status != 0 ||
		!strings.Contains(stdout, `"steps"`) {
		t.Errorf("--server box: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestServerFlagNamesTheServerOfConf(t *testing.T) {
	m := effecttest.NewMachine()
	for _, name := range []string{"ws", "box"} {
		m.Files[conf.ServerPath("/h/.config/ws/mac.conf", name)] = []byte("WS_USER=" + name + "\n")
	}
	m.Files["/h/.config/ws/mac.conf"] = nil
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = withServerDirs(m)
	if status, stdout, _ := runWs(p, "", "client", "macos", "conf", "--server", "box"); status != 0 || !strings.HasPrefix(stdout, "WS_USER='box'\n") {
		t.Errorf("--server box: status = %d, stdout = %q", status, stdout)
	}
	if status, _, stderr := runWs(p, "", "client", "macos", "conf"); status != 1 || !strings.Contains(stderr, "several servers") {
		t.Errorf("no flag: status = %d, stderr = %q", status, stderr)
	}
}
