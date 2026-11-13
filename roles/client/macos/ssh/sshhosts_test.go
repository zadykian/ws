package ssh_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const macConf = "/h/.config/ws/mac.conf"

// withLab adds a server named lab at slot 1, with its own user, beside mac.conf.
func withLab(m *effecttest.Machine, text string) {
	m.Dirs["/h/.config/ws/servers"], m.Dirs["/h/.config/ws/servers/lab"] = 0o755, 0o755
	m.Files[macConf] = nil
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte(text)
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" lab"] = effecttest.Result{}
	}
}

func TestSSHWritesASecondServersBlockFromItsFile(t *testing.T) {
	m := sshMachine()
	withLab(m, "WS_TARGET=cloud\nWS_REMOTE=r.example\nWS_REMOTE_PORT=22\nWS_USER=root\nWS_SLOT=1\n")
	c := sshConf(m, &effecttest.Out{})
	c.MacConf = macConf
	s := c.Step()
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	got := string(m.Files["/h/.ssh/config.d/ws"])
	for _, want := range []string{
		"Host lab\n    HostName 10.99.1.1\n    Port 22\n    User root\n    HostKeyAlias lab\n",
		"Host ws\n    HostName 10.99.0.1\n    Port 22\n    User me\n    HostKeyAlias ws\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("config.d/ws lacks %q:\n%s", want, got)
		}
	}
}

func TestSSHStopsWhereAnotherServersFileFails(t *testing.T) {
	m := sshMachine()
	withLab(m, "WS_SLOT=99\n")
	c := sshConf(m, &effecttest.Out{})
	c.MacConf = macConf
	_, err := c.Step().Plan(context.Background())
	if err == nil || !strings.HasPrefix(err.Error(), "server lab: ") {
		t.Errorf("Plan err = %v, want a refusal that names server lab", err)
	}
}
