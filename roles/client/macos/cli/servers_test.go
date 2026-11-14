package cli

import (
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
