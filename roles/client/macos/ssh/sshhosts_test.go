package ssh_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/values"
)

const macConf = "/h/.config/ws/values.yml"

// withLab adds a server named lab at slot 1, with its own user, beside values.yml.
func withLab(m *effecttest.Machine, text string) {
	m.Dirs["/h/.config/ws/servers"], m.Dirs["/h/.config/ws/servers/lab"] = 0o755, 0o755
	m.Files[macConf] = nil
	m.Files["/h/.config/ws/servers/lab/values.yml"] = []byte(text)
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" lab"] = effecttest.Result{}
	}
}

func TestSSHWritesASecondServersBlockFromItsFile(t *testing.T) {
	m := sshMachine()
	withLab(m, values.Text(
		"WS_TARGET", "cloud",
		"WS_REMOTE", "r.example",
		"WS_REMOTE_PORT", "22",
		"WS_USER", "root",
		"WS_SLOT", "1",
	))
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
	got := string(m.Files["/etc/ws/ssh_config"])
	for _, want := range []string{
		"Host lab\n    HostName 10.99.1.1\n    Port 22\n    User root\n    HostKeyAlias lab\n",
		"Host ws\n    HostName 10.99.0.1\n    Port 22\n    User me\n    HostKeyAlias ws\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("/etc/ws/ssh_config lacks %q:\n%s", want, got)
		}
	}
}

func TestSSHStopsWhereAnotherServersFileFails(t *testing.T) {
	m := sshMachine()
	withLab(m, values.Text("WS_SLOT", "99"))
	c := sshConf(m, &effecttest.Out{})
	c.MacConf = macConf
	_, err := c.Step().Plan(context.Background())
	if err == nil || !strings.HasPrefix(err.Error(), "server lab: ") {
		t.Errorf("Plan err = %v, want a refusal that names server lab", err)
	}
}

func TestSSHStopsWhereSSHReadsAServerAtAnotherAddress(t *testing.T) {
	m := sshMachine()
	withLab(m, values.Text(
		"WS_TARGET", "cloud",
		"WS_REMOTE", "r.example",
		"WS_REMOTE_PORT", "22",
		"WS_USER", "root",
		"WS_SLOT", "1",
	))
	m.Results["ssh -G -F /tmp/x/whole lab"] = effecttest.Result{Out: []byte("user root\nhostname 10.99.7.1\n")}
	c := sshConf(m, &effecttest.Out{})
	c.MacConf = macConf
	_, err := c.Step().Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "ssh reads lab as hostname 10.99.7.1") {
		t.Errorf("Plan err = %v, want a refusal of lab's address", err)
	}
}

func TestSSHStopsWhereABlockNamesASecondServer(t *testing.T) {
	m := sshMachine()
	withLab(m, values.Text(
		"WS_TARGET", "cloud",
		"WS_REMOTE", "r.example",
		"WS_REMOTE_PORT", "22",
		"WS_USER", "root",
		"WS_SLOT", "1",
	))
	m.Files["/h/.ssh/config"] = []byte("Include /etc/ws/ssh_config\nHost lab\n  User x\n")
	c := sshConf(m, &effecttest.Out{})
	c.MacConf = macConf
	_, err := c.Step().Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Host lab, sets its own values for lab:") {
		t.Errorf("Plan err = %v, want a stop at the block for lab", err)
	}
}
