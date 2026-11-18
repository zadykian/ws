package facts

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// env is the process's environment of a user v, with no WS_VALUES.
func env(k string) string {
	return map[string]string{"USER": "v", "HOME": "/Users/v"}[k]
}

// mac is a machine with values.yml and a file for each server, its text by name.
func mac(servers map[string]string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_IDENTITY_PUB_KEY", "~/.ssh/id.pub"))
	m.Dirs["/etc/ws/servers"] = 0o755
	for name, text := range servers {
		m.Dirs["/etc/ws/servers/"+name] = 0o755
		m.Files["/etc/ws/servers/"+name+"/values.yml"] = []byte(text)
	}
	return m
}

func TestReadListsEachServer(t *testing.T) {
	m := mac(map[string]string{
		"ws":  values.Text("WS_SLOT", "0"),
		"lab": values.Text("WS_SLOT", "1", "WS_TARGET", "cloud", "WS_REMOTE", "198.51.100.7", "WS_REMOTE_PORT", "22"),
	})
	f, err := Read(m, env, "/p")
	if err != nil {
		t.Fatal(err)
	}
	if f.User != "v" || f.Home != "/Users/v" || len(f.Servers) != 2 {
		t.Fatalf("facts = %+v, want v, /Users/v and two servers", f)
	}
	lab, ws := f.Servers[0], f.Servers[1]
	if lab.Name != "lab" || lab.Slot != "1" || lab.Remote != "198.51.100.7" || ws.Name != "ws" {
		t.Errorf("servers = %+v, want lab at slot 1, then ws", f.Servers)
	}
	if want := "/Users/v/Library/Application Support/com.mitchellh.ghostty/config.ghostty"; f.Ghostty.Config != want {
		t.Errorf("Ghostty's config = %q, want %q, as none is there", f.Ghostty.Config, want)
	}
	if lab.IdentityPubKey != "~/.ssh/id.pub" {
		t.Errorf("lab's IdentityPubKey = %q, want the Mac's key joined in", lab.IdentityPubKey)
	}
}

func TestReadRefusesAClash(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "1"), "lab": values.Text("WS_SLOT", "1")})
	if _, err := Read(m, env, "/p"); err == nil || !strings.Contains(err.Error(), "WS_SLOT=1") {
		t.Errorf("read = %v, want the clash of the slot", err)
	}
}

func TestReadRefusesABadValue(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "99")})
	if _, err := Read(m, env, "/p"); err == nil || !strings.Contains(err.Error(), "server ws") {
		t.Errorf("read = %v, want server ws refused", err)
	}
}

func TestReadWithoutServers(t *testing.T) {
	f, err := Read(effecttest.NewMachine(), env, "/p")
	if err != nil || f.User != "v" || len(f.Servers) != 0 {
		t.Errorf("read = %+v, %v; want the user and no server", f, err)
	}
}

func TestReadNeedsTheUser(t *testing.T) {
	if _, err := Read(effecttest.NewMachine(), func(string) string { return "" }, "/p"); err == nil {
		t.Error("read with no USER and HOME = nil, want a refusal")
	}
}
