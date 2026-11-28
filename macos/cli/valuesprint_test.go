package cli

import (
	"path"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/values"
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
		{"home", nil, "", 0, "WS_TARGET='lan'\nWS_HOST=''\nWS_SLOT='0'\n", ""},
		{"env", nil, "/e", 0, "WS_TARGET='lan'\nWS_HOST=''\nWS_SLOT='3'\n", ""},
		{"argument", []string{"/a"}, "/e", 0, "WS_TARGET='lan'\nWS_HOST=''\nWS_SLOT='5'\n", ""},
		{"missing", []string{"/none"}, "", 1, "", "ws macos values print: no server in /none/servers: write NAME/values.yml there"},
		{"refused", []string{"/bad"}, "", 1, "", "ws macos values print: /bad/servers/ws/values.yml, line 2: unknown key WS_NOPE"},
		{"lines", []string{"--lines", "/lines"}, "", 0, "WS_TARGET='lan'\nWS_HOST=''\nWS_SLOT='0'\nWS_PUBLIC_KEY=''\nWS_LAN='10.88.0.0/24'\n", ""},
		{"lines of the server's file", []string{"--lines"}, "", 0, "WS_TARGET='lan'\nWS_HOST=''\nWS_SLOT='0'\n", ""},
		{"pair", []string{"/pairs"}, "", 1, "", "WS_LAN and WS_LAN_ADDRESS go together"},
		{"two files", []string{"a", "b"}, "", 2, "", "ws macos values print: accepts at most 1 arg(s), received 2"},
	}
	m := effecttest.NewMachine()
	m.Files["/etc/ws/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "0"))
	m.Files["/e/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "3"))
	m.Files["/a/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "5"))
	m.Files["/lines/servers/ws/values.yml"] = []byte(values.Text("WS_LAN", "10.88.0.0/24"))
	m.Files["/pairs/servers/ws/values.yml"] = []byte(values.Text("WS_LAN", "10.88.0.0/24"))
	m.Files["/bad/servers/ws/values.yml"] = []byte(values.Text("WS_SLOT", "1", "WS_NOPE", "1"))
	withServerDirs(m)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env["WS_VALUES"] = tt.wsConf
			p := withEnv(fakeProcess(), env)
			p.Machine = m
			got, stdout, stderr := runWs(p, append([]string{"macos", "values", "print"}, tt.args...)...)
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
	m.Files["/bad/servers/ws/values.yml"] = []byte(values.Text("WS_NOPE", "1", "WS_NOPE2", "1"))
	p.Machine = withServerDirs(m)
	status, _, stderr := runWs(p, "macos", "values", "print", "/bad")
	want := "ws macos values print: /bad/servers/ws/values.yml, line 1: unknown key WS_NOPE\nws macos values print: /bad/servers/ws/values.yml, line 2: unknown key WS_NOPE2\n"
	if status != 1 || stderr != want {
		t.Errorf("status = %d, stderr = %q; want 1, %q", status, stderr, want)
	}
}
