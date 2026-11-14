package cli

import (
	"path"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// withServerDirs makes a directory of each server file that m holds, as a machine's directory listing needs.
func withServerDirs(m *effecttest.Machine) *effecttest.Machine {
	for f := range m.Files {
		if dir := path.Dir(f); path.Base(f) == "server.conf" {
			m.Dirs[dir], m.Dirs[path.Dir(dir)] = 0o755, 0o755
		}
	}
	return m
}

func TestConfCommand(t *testing.T) {
	env := map[string]string{"HOME": "/h", "WS_CONF": ""}
	tests := []struct {
		name       string
		args       []string
		wsConf     string
		wantStatus int
		wantOut    string
		wantErr    string
	}{
		{"home", nil, "", 0, "WS_USER='ada'\n", ""},
		{"env", nil, "/e/mac.conf", 0, "WS_USER='eve'\n", ""},
		{"argument", []string{"/a/mac.conf"}, "/e/mac.conf", 0, "WS_USER='amy'\n", ""},
		{"missing", []string{"/none"}, "", 1, "", "ws client macos conf: /none is missing: roles/client/macos/README.md"},
		{"refused", []string{"/bad"}, "", 1, "", "ws client macos conf: /bad, line 2: unknown key WS_NOPE"},
		{"lines", []string{"--lines", "/lines/mac.conf"}, "", 0, "WS_USER='zadykian'\nWS_TARGET='lan'\nWS_SLOT='0'\nWS_PUBLIC_KEY=''\nWS_LAN='10.88.0.0/24'\n", ""},
		{"lines of the server's file", []string{"--lines"}, "", 0, "WS_USER='ada'\n", ""},
		{"lines refuse a server's key in mac.conf", []string{"--lines", "/lines/bad/mac.conf"}, "", 1, "", "/lines/bad/mac.conf, line 1: WS_LAN is a server's key"},
		{"pair", []string{"/pairs/mac.conf"}, "", 1, "", "WS_LAN and WS_LAN_ADDRESS go together"},
		{"two files", []string{"a", "b"}, "", 2, "", "ws client macos conf: accepts at most 1 arg(s), received 2"},
	}
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("")
	m.Files["/h/.config/ws/servers/ws/server.conf"] = []byte("WS_USER=ada\n")
	m.Files["/e/mac.conf"] = []byte("")
	m.Files["/e/servers/ws/server.conf"] = []byte("WS_USER=eve\n")
	m.Files["/a/mac.conf"] = []byte("")
	m.Files["/a/servers/ws/server.conf"] = []byte("WS_USER=amy\n")
	m.Files["/lines/mac.conf"] = []byte("WS_GIT_NAME=zed\n")
	m.Files["/lines/servers/ws/server.conf"] = []byte("WS_USER=zadykian\nWS_LAN=10.88.0.0/24\n")
	m.Files["/lines/bad/mac.conf"] = []byte("WS_LAN=10.88.0.0/24\n")
	m.Files["/lines/bad/servers/ws/server.conf"] = []byte("")
	m.Files["/pairs/mac.conf"] = []byte("")
	m.Files["/pairs/servers/ws/server.conf"] = []byte("WS_LAN=10.88.0.0/24\n")
	m.Files["/bad"] = []byte("WS_GIT_NAME=ada\nWS_NOPE=1\n")
	m.Files["/servers/ws/server.conf"] = []byte("")
	withServerDirs(m)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env["WS_CONF"] = tt.wsConf
			p := withEnv(fakeProcess(), env)
			p.Machine = m
			got, stdout, stderr := runWs(p, "", append([]string{"client", "macos", "conf"}, tt.args...)...)
			if got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if !strings.HasPrefix(stdout, tt.wantOut) {
				t.Errorf("stdout = %q, want it to start with %q", stdout, tt.wantOut)
			}
			if !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.wantErr)
			}
		})
	}
}

func TestConfReportsEachLineOfARefusedFile(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{})
	m := effecttest.NewMachine()
	m.Files["/bad"] = []byte("WS_NOPE=1\nWS_NOPE2=1\n")
	m.Files["/servers/ws/server.conf"] = []byte("")
	p.Machine = withServerDirs(m)
	status, _, stderr := runWs(p, "", "client", "macos", "conf", "/bad")
	want := "ws client macos conf: /bad, line 1: unknown key WS_NOPE\nws client macos conf: /bad, line 2: unknown key WS_NOPE2\n"
	if status != 1 || stderr != want {
		t.Errorf("status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
}
