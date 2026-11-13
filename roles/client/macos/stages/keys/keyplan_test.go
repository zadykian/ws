package keys

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/setup"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

const (
	socket   = "/h/" + sshcfg.SecretiveData + "/socket.ssh"
	listKeys = "env SSH_AUTH_SOCK=" + socket + " ssh-add -L"
)

// agent gives the Mac Secretive's socket, whose agent lists the lines.
func agent(m *effecttest.Machine, lines ...string) {
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results[listKeys] = effecttest.Result{Out: []byte(strings.Join(lines, ""))}
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"], m.Dirs["/h/"+sshcfg.KeyCopies] = 0o755, 0o700, 0o755
}

// copyOf is the path of ws's copy of the line's key.
func copyOf(line string) string { return sshcfg.Expand(sshcfg.AgentKeys(line)[0].Pub(), "/h") }

func planned(k *MacKey) string {
	p, err := k.Plan(context.Background())
	return fmt.Sprint(p.Changes, err)
}

func TestMacKeyPlan(t *testing.T) {
	const conf = "WS_IDENTITY_PUB_KEY=~/k.pub\nWS_IDENTITY_AGENT=/s\n"
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
	if got := planned(k); got != "[Mac key: would ask for ws's login key, in Secretive: make a new one (the default)] <nil>" {
		t.Errorf("no key: Plan = %q", got)
	}
	k, _, _ = newMacKey(t, gitConf)
	if got := planned(k); !strings.Contains(got, "for ws's login key") || !strings.Contains(got, "for git's signing key") {
		t.Errorf("no key and git's name: Plan = %q, want both questions", got)
	}
}

// A file key that mac.conf names must be there: the stage makes none.
func TestMacKeyPlanNeedsAFileKey(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub\n")
	if got := planned(k); !strings.Contains(got, "/h/.ssh/id_ed25519.pub, named in mac.conf's WS_IDENTITY_PUB_KEY, is missing") {
		t.Errorf("Plan of a missing key = %q", got)
	}
	m.Files["/h/.ssh/id_ed25519"], m.Files["/h/.ssh/id_ed25519.pub"] = []byte("p"), []byte(pubText)
	m.Results["ln -sf /h/.ssh/id_ed25519 "+lockDir+"/key"] = effecttest.Result{}
	m.Results["rm -f "+lockDir+"/key"] = effecttest.Result{}
	m.Results["ssh-keygen -l -f "+lockDir+"/key"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	if got := planned(k); got != "[] <nil>" {
		t.Errorf("Plan of the key there = %q", got)
	}
}

// macOS 27 denies ws the files of Secretive's container, as #839 of Secretive's says. So the stage
// must offer the agent's keys alone, and plan the copy of the one named that the stages after it
// read.
func TestMacKeyPlanTakesTheAgentsKey(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m, p256)
	if got := planned(k); !strings.Contains(got, ": "+p256Label+"; make a new one (the default)]") {
		t.Errorf("Plan = %q, want the agent's key offered", got)
	}
	m.Files[confPathForTest] = []byte("WS_IDENTITY_PUB_KEY=" + listed(p256)[0].Pub() + "\n")
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

// git's question marks ws's login key, which mac.conf names. An empty WS_GIT_SIGNING_PUB_KEY names none.
func TestMacKeyPlanAsksForGitsKey(t *testing.T) {
	k, m, _ := newMacKey(t, gitConf+"WS_GIT_SIGNING_PUB_KEY=\n")
	agent(m, p256)
	if got := planned(k); strings.Count(got, "would ask for") != 2 {
		t.Errorf("Plan with neither key = %q, want both questions", got)
	}
	m.Files[confPathForTest] = []byte(gitConf + "WS_IDENTITY_PUB_KEY=" + listed(p256)[0].Pub() + "\n")
	m.Files[copyOf(p256)] = []byte(p256)
	want := "[Mac key: would ask for git's signing key, in Secretive: " + p256Label + ", ws's login key; " +
		"make a new one (the default)] <nil>"
	if got := planned(k); got != want {
		t.Errorf("Plan = %q, want %q", got, want)
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
