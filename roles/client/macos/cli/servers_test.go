package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// serversFixture is ws on the LAN with a running tunnel, and lab in the cloud with none.
func serversFixture() (*effecttest.Machine, func(...string) (int, string, string)) {
	p := withEnv(fakeProcess(), map[string]string{"WS_CONF": "/e/mac.conf"})
	m := p.Machine.(*effecttest.Machine)
	m.Files["/e/mac.conf"] = []byte("")
	m.Files["/e/servers/ws/server.conf"] = []byte("WS_USER=ada\n")
	m.Files["/e/servers/lab/server.conf"] = []byte("WS_TARGET=cloud\nWS_SLOT=1\nWS_REMOTE=192.0.2.9\nWS_REMOTE_PORT=22\n")
	withServerDirs(m)
	m.Files["/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist"] = []byte("plist")
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	m.Files["/var/run/ws-tunnel.ws.state"] = []byte("tier=lan\nendpoint=10.88.0.10:51820\ninterface=utun5\nsince=1\nroutes=\nleft_out=\n")
	return m, func(args ...string) (int, string, string) {
		return runWs(p, "", append([]string{"client", "macos", "servers"}, args...)...)
	}
}

func TestServersListsEachServerWithItsTunnel(t *testing.T) {
	_, run := serversFixture()
	status, stdout, stderr := run()
	want := "NAME  TARGET  SLOT  TUNNEL\n" +
		"lab   cloud   1     absent\n" +
		"ws    lan     0     lan 10.88.0.10:51820\n"
	if status != 0 || stdout != want || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0, %q", status, stdout, stderr, want)
	}
}

func TestServersShowsAStoppedTunnel(t *testing.T) {
	m, run := serversFixture()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{Err: errors.New("not loaded")}
	if _, stdout, _ := run(); !strings.Contains(stdout, "ws    lan     0     stopped\n") {
		t.Errorf("stdout = %q, want ws stopped", stdout)
	}
}

func TestServersRefusesAServerFlag(t *testing.T) {
	_, run := serversFixture()
	status, stdout, stderr := run("--server", "ws")
	if want := "ws client macos servers: servers lists every server: leave out --server\n"; status != 1 || stdout != "" || stderr != want {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 1, %q", status, stdout, stderr, want)
	}
}

func TestServersNamesTheFileItRefuses(t *testing.T) {
	m, run := serversFixture()
	m.Files["/e/servers/lab/server.conf"] = []byte("WS_TARGET=moon\n")
	status, stdout, stderr := run()
	if status != 1 || !strings.Contains(stdout, "lab   -       -     absent\n") || !strings.Contains(stdout, "ws    lan     0     lan ") {
		t.Errorf("status = %d, stdout = %q; want 1 with every row, lab as -", status, stdout)
	}
	if !strings.Contains(stderr, "/e/servers/lab/server.conf") || !strings.Contains(stderr, "WS_TARGET") {
		t.Errorf("stderr = %q, want lab's errors", stderr)
	}
}

func TestServersWithNoneSaysHowToAddOne(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"WS_CONF": "/e/mac.conf"})
	status, _, stderr := runWs(p, "", "client", "macos", "servers")
	if status != 1 || !strings.Contains(stderr, "no server beside /e/mac.conf: ws client macos wizard adds one") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestServersSaysAMissingMacConfOnce(t *testing.T) {
	m, run := serversFixture()
	delete(m.Files, "/e/mac.conf")
	_, _, stderr := run()
	if n := strings.Count(stderr, "/e/mac.conf is missing"); n != 1 {
		t.Errorf("stderr says mac.conf is missing %d times, want once: %q", n, stderr)
	}
}
