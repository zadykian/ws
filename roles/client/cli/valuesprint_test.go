package cli

import (
	"path"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/values"
)

// withServerDirs makes a directory of each server file that m holds, as a machine's directory listing needs.
func withServerDirs(m *effecttest.Machine) *effecttest.Machine {
	for f := range m.Files {
		if dir := path.Dir(f); path.Base(f) == "values.yml" {
			m.Dirs[dir], m.Dirs[path.Dir(dir)] = 0o755, 0o755
		}
	}
	return m
}

func TestValuesPrintCommand(t *testing.T) {
	env := map[string]string{"HOME": "/h", "WS_VALUES": ""}
	tests := []struct {
		name       string
		args       []string
		wsConf     string
		wantStatus int
		wantOut    string
		wantErr    string
	}{
		{"home", nil, "", 0, "WS_TARGET='lan'\nWS_SLOT='0'\n", ""},
		{"env", nil, "/e/values.yml", 0, "WS_TARGET='lan'\nWS_SLOT='3'\n", ""},
		{"argument", []string{"/a/values.yml"}, "/e/values.yml", 0, "WS_TARGET='lan'\nWS_SLOT='5'\n", ""},
		{"missing", []string{"/none"}, "", 1, "", "ws client values print: /none is missing: roles/client/values/README.md"},
		{"refused", []string{"/bad"}, "", 1, "", "ws client values print: /bad, line 2: unknown key WS_NOPE"},
		{"lines", []string{"--lines", "/lines/values.yml"}, "", 0, "WS_TARGET='lan'\nWS_SLOT='0'\nWS_PUBLIC_KEY=''\nWS_LAN='10.88.0.0/24'\n", ""},
		{"lines of the server's file", []string{"--lines"}, "", 0, "WS_TARGET='lan'\nWS_SLOT='0'\n", ""},
		{"lines refuse a server's key in values.yml", []string{"--lines", "/lines/bad/values.yml"}, "", 1, "", "/lines/bad/values.yml, line 1: WS_LAN is a server's key"},
		{"pair", []string{"/pairs/values.yml"}, "", 1, "", "WS_LAN and WS_LAN_ADDRESS go together"},
		{"two files", []string{"a", "b"}, "", 2, "", "ws client values print: accepts at most 1 arg(s), received 2"},
	}
	m := effecttest.NewMachine()
	m.Files["/etc/ws/values.yml"] = []byte("")
	m.Files["/etc/ws/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "0"))
	m.Files["/e/values.yml"] = []byte("")
	m.Files["/e/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "3"))
	m.Files["/a/values.yml"] = []byte("")
	m.Files["/a/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "5"))
	m.Files["/lines/values.yml"] = []byte(values.Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub"))
	m.Files["/lines/servers/ws/values.yml"] = []byte(values.Text("WS_LAN", "10.88.0.0/24"))
	m.Files["/lines/bad/values.yml"] = []byte(values.Text("WS_LAN", "10.88.0.0/24"))
	m.Files["/lines/bad/servers/ws/values.yml"] = []byte("")
	m.Files["/pairs/values.yml"] = []byte("")
	m.Files["/pairs/servers/ws/values.yml"] = []byte(values.Text("WS_LAN", "10.88.0.0/24"))
	m.Files["/bad"] = []byte(values.Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub", "WS_NOPE", "1"))
	m.Files["/servers/ws/values.yml"] = []byte("")
	withServerDirs(m)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env["WS_VALUES"] = tt.wsConf
			p := withEnv(fakeProcess(), env)
			p.Machine = m
			got, stdout, stderr := runWs(p, append([]string{"client", "values", "print"}, tt.args...)...)
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

func TestValuesPrintReportsEachLineOfARefusedFile(t *testing.T) {
	p := withEnv(fakeProcess(), map[string]string{})
	m := effecttest.NewMachine()
	m.Files["/bad"] = []byte(values.Text("WS_NOPE", "1", "WS_NOPE2", "1"))
	m.Files["/servers/ws/values.yml"] = []byte("")
	p.Machine = withServerDirs(m)
	status, _, stderr := runWs(p, "client", "values", "print", "/bad")
	want := "ws client values print: /bad, line 1: unknown key WS_NOPE\nws client values print: /bad, line 2: unknown key WS_NOPE2\n"
	if status != 1 || stderr != want {
		t.Errorf("status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
}
