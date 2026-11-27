package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/values"
)

// serversFixture is ws on the LAN with a running tunnel, and lab in the cloud with none.
func serversFixture() (*effecttest.Machine, func(...string) (int, string, string)) {
	p := withEnv(fakeProcess(), map[string]string{"WS_VALUES": "/e/values.yml"})
	m := p.Machine.(*effecttest.Machine)
	m.Files["/e/values.yml"] = []byte("")
	m.Files["/e/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "0"))
	m.Files["/e/servers/lab/values.yml"] = []byte(values.Text(
		"WS_TARGET", "cloud",
		"WS_SLOT", "1",
		"WS_REMOTE", "192.0.2.9",
		"WS_REMOTE_PORT", "22",
	))
	withServerDirs(m)
	m.Files["/Library/LaunchDaemons/com.github.zadykian.ws-tunnel.ws.plist"] = []byte("plist")
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel.ws"] = effecttest.Result{}
	m.Files["/var/run/ws-tunnel.ws.state"] = []byte("tier=lan\nendpoint=10.88.0.10:51820\ninterface=utun5\nsince=1\nroutes=\nleft_out=\n")
	return m, func(args ...string) (int, string, string) {
		return runWs(p, append([]string{"macos", "servers"}, args...)...)
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

func TestServersListsAGuestByItsTarget(t *testing.T) {
	m, run := serversFixture()
	m.Files["/e/servers/g1/values.yml"] = []byte(values.Text("WS_HOST", "ws", "WS_SLOT", "2", "WS_TUNNEL_PORT", "51822"))
	withServerDirs(m)
	_, stdout, stderr := run()
	if want := "g1    guest   2     absent\n"; !strings.Contains(stdout, want) || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q; want it to hold %q", stdout, stderr, want)
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
	if want := "ws macos servers: servers lists every server: leave out --server\n"; status != 1 || stdout != "" || stderr != want {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 1, %q", status, stdout, stderr, want)
	}
}

func TestServersNamesTheFileItRefuses(t *testing.T) {
	m, run := serversFixture()
	m.Files["/e/servers/lab/values.yml"] = []byte(values.Text("WS_TARGET", "moon"))
	status, stdout, stderr := run()
	if status != 1 || !strings.Contains(stdout, "lab   -       -     absent\n") || !strings.Contains(stdout, "ws    lan     0     lan ") {
		t.Errorf("status = %d, stdout = %q; want 1 with every row, lab as -", status, stdout)
	}
	if !strings.Contains(stderr, "/e/servers/lab/values.yml") || !strings.Contains(stderr, "WS_TARGET") {
		t.Errorf("stderr = %q, want lab's errors", stderr)
	}
}

func TestServersWithNoneSaysHowToAddOne(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{"WS_VALUES": "/e/values.yml"})
	p.Machine.(*effecttest.Machine).Files["/e/values.yml"] = []byte("")
	status, _, stderr := runWs(p, "macos", "servers")
	if status != 1 || !strings.Contains(stderr, "no server beside /e/values.yml: write NAME/values.yml there, as macos/README.md says") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestServersSaysAMissingMacConfOnce(t *testing.T) {
	m, run := serversFixture()
	delete(m.Files, "/e/values.yml")
	_, _, stderr := run()
	if n := strings.Count(stderr, "/e/values.yml is missing"); n != 1 {
		t.Errorf("stderr says values.yml is missing %d times, want once: %q", n, stderr)
	}
}

func TestServersRefusesABadMacFileOnce(t *testing.T) {
	m, run := serversFixture()
	m.Files["/e/values.yml"] = []byte(values.Text("WS_SLOT", "3"))
	status, stdout, stderr := run()
	if status != 1 || stdout != "" || strings.Count(stderr, "/e/values.yml") != 1 || strings.Contains(stderr, "fix its file") {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 1 and one refusal of values.yml alone", status, stdout, stderr)
	}
}

// TestServersRefusesABadMacFileBeforeABadName checks that values.yml is refused first and alone,
// where servers/ holds a bad name or no server, as the values page says.
func TestServersRefusesABadMacFileBeforeABadName(t *testing.T) {
	for name, set := range map[string]func(m *effecttest.Machine){
		"a bad name": func(m *effecttest.Machine) {
			m.Files["/e/servers/Lab/values.yml"] = []byte(values.Text("WS_SLOT", "2"))
			withServerDirs(m)
		},
		"no server": func(m *effecttest.Machine) {
			for f := range m.Files {
				if strings.HasPrefix(f, "/e/servers/") {
					delete(m.Files, f)
				}
			}
		},
	} {
		m, run := serversFixture()
		m.Files["/e/values.yml"] = []byte(values.Text("WS_SLOT", "3"))
		set(m)
		status, stdout, stderr := run()
		if status != 1 || stdout != "" || strings.Count(stderr, "/e/values.yml") != 1 ||
			strings.Contains(stderr, "not a server's name") || strings.Contains(stderr, "no server beside") {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q; want 1 and values.yml's refusal alone", name, status,
				stdout, stderr)
		}
	}
}
