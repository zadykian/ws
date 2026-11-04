package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
)

const (
	socket   = "/h/" + sshcfg.SecretiveData + "/socket.ssh"
	listKeys = "env SSH_AUTH_SOCK=" + socket + " ssh-add -L"
	makeDir  = "mkdir -p /h/" + sshcfg.KeyCopies
)

// agent gives the Mac Secretive's socket, whose agent lists the lines.
func agent(m *effecttest.Machine, lines ...string) {
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results[listKeys] = effecttest.Result{Out: []byte(strings.Join(lines, ""))}
	m.Results[makeDir] = effecttest.Result{}
}

// copyOf is the path of ws's copy of the line's key.
func copyOf(line string) string { return sshcfg.Expand(sshcfg.AgentKeys(line)[0].Pub(), "/h") }

func planned(k *MacKey) string {
	p, err := k.Plan(context.Background())
	return fmt.Sprint(p.Changes, err)
}

func TestMacKeyPlan(t *testing.T) {
	const conf = "WS_IDENTITY=~/k.pub\nWS_IDENTITY_AGENT=/s\n"
	k, m, _ := newMacKey(t, conf)
	m.Files["/h/k.pub"] = []byte(pubText)
	m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["env SSH_AUTH_SOCK=/s ssh-add -L"] = effecttest.Result{Out: []byte("ssh-ed25519 AAAAC3 x\n")}
	if got := planned(k); got != "[] <nil>" {
		t.Errorf("a listed key: Plan = %q", got)
	}
	m.Results["env SSH_AUTH_SOCK=/s ssh-add -L"] = effecttest.Result{Out: []byte("ssh-rsa BBBB x\n")}
	if got := planned(k); !strings.Contains(got, "the agent at /s lists no key of /h/k.pub") {
		t.Errorf("a key the agent lacks: Plan = %q", got)
	}
	k, _, _ = newMacKey(t, "")
	if got := planned(k); !strings.Contains(got, "would ask whether Secretive keeps the Mac's key") {
		t.Errorf("no key: Plan = %q", got)
	}
}

func TestMacKeyPlanPinsAFileKey(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_REMOTE=\n")
	m.Files["/h/.ssh/id_ed25519"], m.Files["/h/.ssh/id_ed25519.pub"] = []byte("p"), []byte(pubText)
	m.Results["ln -sf /h/.ssh/id_ed25519 "+lockDir+"/key"] = effecttest.Result{}
	m.Results["rm -f "+lockDir+"/key"] = effecttest.Result{}
	m.Results["ssh-keygen -l -f "+lockDir+"/key"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	if got := planned(k); !strings.Contains(got, "\n+WS_IDENTITY=~/.ssh/id_ed25519.pub\n") {
		t.Errorf("Plan = %q, want the line for the file key", got)
	}
}

// macOS 27 denies ws the files of Secretive's container, as #839 of Secretive's says: the stage must
// take the key from the agent alone, and plan the copy that the stages after it read.
func TestMacKeyPlanTakesTheAgentsKey(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m, p256)
	p, err := k.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 || p.Changes[0] != (setup.KeyCopy{Path: copyOf(p256), Text: p256}) {
		t.Fatalf("Plan = %v, %v; want the copy alone", p.Changes, err)
	}
	for _, call := range m.Calls {
		if strings.Contains(call, "PublicKeys") {
			t.Errorf("Plan touched Secretive's container: %s", call)
		}
	}
	if err := k.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := planned(k); got != "[] <nil>" {
		t.Errorf("Plan with the copy there = %q", got)
	}
}

func TestMacKeyPlanWaitsForGitsKey(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_GIT_NAME=ada\nWS_GIT_EMAIL=a@b.c\n")
	agent(m, p256)
	m.Files[copyOf(p256)] = []byte(p256)
	if got := planned(k); !strings.Contains(got, "wait for git's signing key made there") {
		t.Errorf("Plan = %q", got)
	}
}

func TestMacKeyPlanStopsWhereTheAgentIsDown(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m)
	m.Results[listKeys] = effecttest.Result{Err: errors.New("exit status 2: Connection refused")}
	if got := planned(k); !strings.Contains(got, "does not answer") {
		t.Errorf("Plan = %q", got)
	}
}
