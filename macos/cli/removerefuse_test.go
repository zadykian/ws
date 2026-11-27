package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/macos/values"
)

func TestRemoveRefusesAHostThatHasGuests(t *testing.T) {
	p, m := removeFixture()
	m.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_HOST", "ws", "WS_SLOT", "1"))
	status, _, stderr := runWs(p, "macos", "remove", "ws")
	if want := "ws is the host of lab: remove them first"; status != 1 || !strings.Contains(stderr, want) {
		t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, want)
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "run rm") {
			t.Errorf("a refused remove ran %q", call)
		}
	}
}

func TestRemoveRefusesTheLastServerAndAWrongName(t *testing.T) {
	for name, tc := range map[string]struct {
		arg, want string
		last      bool
	}{
		"last server":  {"ws", "ws is the Mac's one server: with none, the bootstrap prunes no daemon, so remove leaves it", true},
		"unknown name": {"nope", "no server named nope in /etc/ws/servers, which holds lab and ws", false},
		"bad name":     {"Bad_Name", "Bad_Name is not a server's name: " + values.NameForm, false},
		"path":         {"../ws", "../ws is not a server's name: " + values.NameForm, false},
	} {
		t.Run(name, func(t *testing.T) {
			p, m := removeFixture()
			if tc.last {
				delete(m.Files, "/etc/ws/servers/lab/values.yml")
				delete(m.Dirs, "/etc/ws/servers/lab")
			}
			status, _, stderr := runWs(p, "macos", "remove", tc.arg)
			if status != 1 || !strings.Contains(stderr, tc.want) || strings.Contains(stderr, "--server") {
				t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, tc.want)
			}
			for _, call := range m.Calls {
				if strings.HasPrefix(call, "run rm") {
					t.Errorf("a refused remove ran %q", call)
				}
			}
		})
	}
}

func TestRemoveRefusesAServerFlag(t *testing.T) {
	p, _ := removeFixture()
	status, _, stderr := runWs(p, "macos", "--server", "ws", "remove", "lab")
	if status != 1 || !strings.Contains(stderr, "remove takes the server's name alone") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
