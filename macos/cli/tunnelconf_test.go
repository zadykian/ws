package cli

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/values"
)

// labelOf is the label of a server's daemon.
func labelOf(name string) string { return "com.github.zadykian.ws-tunnel." + name }

// keyedMac is a Mac with a server ws whose file has WS_PUBLIC_KEY, and a directory in
// /etc/ws/tunnel for each of names.
func keyedMac(names ...string) *effecttest.Machine {
	m := effecttest.NewMachine()
	writeConfs(m, values.Text("WS_SLOT", "0", "WS_PUBLIC_KEY", wgKey, "WS_REMOTE", "198.51.100.7"))
	if len(names) > 0 {
		m.Dirs["/etc/ws/tunnel"] = 0o755
		m.Root["/etc/ws/tunnel"] = effect.Attrs{Mode: 0o755}
	}
	for _, name := range names {
		m.Dirs["/etc/ws/tunnel/"+name] = 0o755
		m.Root["/etc/ws/tunnel/"+name] = effect.Attrs{Mode: 0o755}
	}
	return m
}

func TestTunnelConfWritesEachServer(t *testing.T) {
	m := keyedMac("ws")
	run := partEnv(m)
	if status, stdout, stderr := run("tunnel", "conf", "--check"); status != 1 || stdout != "" ||
		!strings.Contains(stderr, "/etc/ws/tunnel/ws/tunnel.conf") {
		t.Errorf("tunnel conf --check: status = %d, stdout = %q, stderr = %q; want 1, tunnel.conf due", status, stdout, stderr)
	}
	if status, _, stderr := run("tunnel", "conf"); status != 0 {
		t.Fatalf("tunnel conf: status = %d, stderr = %q; want 0", status, stderr)
	}
	conf := string(m.Files["/etc/ws/tunnel/ws/tunnel.conf"])
	if !strings.Contains(conf, "WS_PUBLIC_KEY="+wgKey+"\n") || !strings.Contains(conf, "WS_REMOTE=198.51.100.7\n") {
		t.Errorf("tunnel.conf = %q, want the server's key and remote", conf)
	}
	if mode := m.Modes["/etc/ws/tunnel/ws/tunnel.conf"]; mode != 0o644 {
		t.Errorf("tunnel.conf's mode = %o, want 644, readable by all", mode)
	}
	if _, ok := m.Dirs["/etc/ws/tunnel/ws/routes.d"]; !ok {
		t.Errorf("dirs = %v, want routes.d", m.Dirs)
	}
	if status, _, stderr := run("tunnel", "conf", "--check"); status != 0 {
		t.Errorf("second tunnel conf --check: status = %d, stderr = %q; want 0", status, stderr)
	}
}

func TestTunnelConfLeavesAServerWithNoKey(t *testing.T) {
	m := effecttest.NewMachine()
	writeConfs(m, values.Text("WS_SLOT", "0"))
	if status, _, stderr := partEnv(m)("tunnel", "conf"); status != 0 {
		t.Fatalf("tunnel conf: status = %d, stderr = %q; want 0", status, stderr)
	}
	if _, ok := m.Dirs["/etc/ws/tunnel"]; ok {
		t.Errorf("dirs = %v, want no /etc/ws/tunnel without WS_PUBLIC_KEY", m.Dirs)
	}
}

func TestTunnelConfPrunesAGoneServer(t *testing.T) {
	m := keyedMac("lab", "ws")
	m.Dirs["/Library/LaunchDaemons"], m.Dirs["/var/log"] = 0o755, 0o755
	for _, name := range []string{"lab", "ws"} {
		m.Files["/Library/LaunchDaemons/"+labelOf(name)+".plist"] = []byte("x")
		m.Results["launchctl print system/"+labelOf(name)] = effecttest.Result{}
	}
	m.Files["/var/log/ws-tunnel.lab.log"] = []byte("x")
	m.Results["launchctl bootout system/"+labelOf("lab")] = effecttest.Result{}
	// The bootout waits until print no longer finds lab, before its files go.
	gone := effecttest.Result{Err: exec.Command("sh", "-c", "exit 113").Run()}
	m.After = map[string]map[string]effecttest.Result{
		"launchctl bootout system/" + labelOf("lab"): {"launchctl print system/" + labelOf("lab"): gone},
	}
	m.Results["launchctl kill SIGHUP system/"+labelOf("ws")] = effecttest.Result{}
	if status, _, stderr := partEnv(m)("tunnel", "conf"); status != 0 {
		t.Fatalf("tunnel conf: status = %d, stderr = %q; want 0", status, stderr)
	}
	var root []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root launchctl") || strings.HasPrefix(c, "root rm") {
			root = append(root, c)
		}
	}
	want := []string{
		"root launchctl bootout system/" + labelOf("lab"),
		"root rm -rf -- /etc/ws/tunnel/lab",
		"root rm -rf -- /var/log/ws-tunnel.lab.log",
		"root launchctl kill SIGHUP system/" + labelOf("ws"),
	}
	if !slices.Equal(root, want) {
		t.Errorf("root's calls = %q, want %q", root, want)
	}
	// launchd's unit takes lab's plist away after this one: until then, a run has nothing to do.
	if status, _, stderr := partEnv(m)("tunnel", "conf", "--check"); status != 0 {
		t.Errorf("tunnel conf --check with lab's plist alone: status = %d, stderr = %q; want 0", status, stderr)
	}
}

func TestTunnelConfPrunesNothingWithNoServer(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/ws/tunnel"], m.Dirs["/etc/ws/tunnel/ws"] = 0o755, 0o755
	m.Results["launchctl print system/"+labelOf("ws")] = effecttest.Result{}
	if status, _, stderr := partEnv(m)("tunnel", "conf", "--check"); status != 0 {
		t.Errorf("tunnel conf --check: status = %d, stderr = %q; want 0, a listing with no server is not this Mac's", status, stderr)
	}
}

func TestTunnelKeyNeedsRoot(t *testing.T) {
	if status, _, stderr := partEnv(effecttest.NewMachine())("tunnel", "key"); status != 1 || !strings.Contains(stderr, "as root") {
		t.Errorf("tunnel key: status = %d, stderr = %q; want 1, root asked for", status, stderr)
	}
}
