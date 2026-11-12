package setup_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

const (
	agentHome = "/h"
	agentData = agentHome + "/" + sshcfg.SecretiveData
	agentSock = agentData + "/socket.ssh"
	agentList = "env SSH_AUTH_SOCK=" + agentSock + " ssh-add -L"
	agentKey  = "ecdsa-sha2-nistp256 AAAA me\n"
	agentGit  = "ecdsa-sha2-nistp256 BBBB me\n"
)

// agentMachine has Secretive's socket, whose agent lists the lines.
func agentMachine(m *effecttest.Machine, lines ...string) *effecttest.Machine {
	m.Files[agentSock], m.Modes[agentSock] = nil, fs.ModeSocket
	m.Results[agentList] = effecttest.Result{Out: []byte(strings.Join(lines, ""))}
	return m
}

// copyOf is the path of ws's copy of the line's key.
func copyOf(line string) string { return sshcfg.Expand(sshcfg.AgentKeys(line)[0].Pub(), agentHome) }

func TestSecretiveKeys(t *testing.T) {
	ctx := context.Background()
	m := effecttest.NewMachine()
	if keys, socket, err := setup.SecretiveKeys(ctx, m, agentHome); keys != nil || socket || err != nil {
		t.Errorf("no socket: %v, %v, %v", keys, socket, err)
	}
	agentMachine(m, agentKey, agentGit)
	if keys, socket, err := setup.SecretiveKeys(ctx, m, agentHome); len(keys) != 2 || !socket || err != nil {
		t.Errorf("two keys: %v, %v, %v", keys, socket, err)
	}
	// ssh-add's words for no keys, with its exit 1.
	m.Results[agentList] = effecttest.Result{Out: []byte("The agent has no identities.\n"), Err: errors.New("exit status 1")}
	if keys, socket, err := setup.SecretiveKeys(ctx, m, agentHome); len(keys) != 0 || !socket || err != nil {
		t.Errorf("an empty agent: %v, %v, %v", keys, socket, err)
	}
	// The socket stays after Secretive quits.
	m.Results[agentList] = effecttest.Result{Err: errors.New("exit status 2: Connection refused")}
	if _, socket, err := setup.SecretiveKeys(ctx, m, agentHome); !socket || !errors.Is(err, setup.ErrAgentDown) {
		t.Errorf("a stale socket: %v, %v", socket, err)
	}
}

// macOS 27 denies ws every file in Secretive's container: ReadKey must never need one.
func TestReadKeyTakesTheAgentsKeyWithNoFile(t *testing.T) {
	m := agentMachine(effecttest.NewMachine(), agentKey)
	pub := sshcfg.AgentKeys(agentKey)[0].Pub()
	id, err := setup.ReadKey(context.Background(), m, conf.Conf{IdentityPubKey: pub}, agentHome, setup.LoginKey)
	if err != nil {
		t.Fatal(err)
	}
	if id.Pub != pub || id.File != copyOf(agentKey) || id.Text != agentKey ||
		id.Agent != "~/"+sshcfg.SecretiveData+"/socket.ssh" {
		t.Errorf("ReadKey = %+v", id)
	}
	for _, call := range m.Calls {
		if strings.Contains(call, "PublicKeys") {
			t.Errorf("ReadKey touched Secretive's container: %s", call)
		}
	}
}

func TestPlanCopy(t *testing.T) {
	m := effecttest.NewMachine()
	pub := copyOf(agentKey)
	for name, args := range map[string][2]string{"a .pub elsewhere": {"/h/.ssh/k.pub", agentKey}, "no text": {pub, ""}} {
		if got, err := setup.PlanCopy(m, args[0], args[1], agentHome); got != nil || err != nil {
			t.Errorf("%s: %v, %v", name, got, err)
		}
	}
	got, err := setup.PlanCopy(m, pub, agentKey, agentHome)
	if err != nil || len(got) != 4 || got[2] != (setup.MakeDir{Path: "/h/" + sshcfg.KeyCopies, Mode: 0o755}) ||
		got[3] != (setup.KeyCopy{Path: pub, Text: agentKey}) {
		t.Fatalf("a missing copy: %v, %v; want the directories, then the copy", got, err)
	}
	m.Results["mkdir -p /h/"+sshcfg.KeyCopies] = effecttest.Result{}
	for _, c := range got {
		if err := setup.Apply(context.Background(), m, &effecttest.Out{}, c); err != nil {
			t.Fatal(err)
		}
	}
	if string(m.Files[pub]) != agentKey {
		t.Errorf("the copy holds %q", m.Files[pub])
	}
	if got, err := setup.PlanCopy(m, pub, agentKey, agentHome); got != nil || err != nil {
		t.Errorf("a copy that holds the key: %v, %v", got, err)
	}
}

func TestSSHWritesTheCopyItNames(t *testing.T) {
	m, out := agentMachine(sshMachine(), agentKey), &effecttest.Out{}
	m.Results["mkdir -p /h/.config/ws/secretive"] = effecttest.Result{}
	s := sshStep(m, out)
	s.Conf.IdentityPubKey = sshcfg.AgentKeys(agentKey)[0].Pub()
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[copyOf(agentKey)]) != agentKey {
		t.Errorf("the copy holds %q", m.Files[copyOf(agentKey)])
	}
	if got := string(m.Files["/h/.ssh/config.d/ws"]); !strings.Contains(got, "IdentityFile "+sshcfg.AgentKeys(agentKey)[0].Pub()+"\n") {
		t.Errorf("config.d/ws names no copy: %s", got)
	}
	if again, err := s.Plan(context.Background()); err != nil || len(again.Changes) != 0 {
		t.Errorf("second Plan = %v, %v; want none", again.Changes, err)
	}
}

func TestGitSignsWithTheCopyOfTheAgentsKey(t *testing.T) {
	m, g := gitReady()
	home := g.Home
	data := home + "/" + sshcfg.SecretiveData
	line := agentGit
	socket := data + "/socket.ssh"
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{Out: []byte(line)}
	key := sshcfg.AgentKeys(line)[0]
	pub := sshcfg.Expand(key.Pub(), home)
	// A mac.conf from before macOS 27, whose .pub ws may not read.
	in := data + "/PublicKeys/" + key.Name
	m.Root[in] = m.Root[socket]
	g.Conf.GitSigningPubKey = in
	p, err := g.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var copied, config bool
	for _, c := range p.Changes {
		copied = copied || c == setup.KeyCopy{Path: pub, Text: line}
		config = config || strings.Contains(c.String(), `signingkey = "`+pub+`"`)
	}
	if !copied || !config {
		t.Errorf("Plan = %v; want ws's copy, named as git's key", p.Changes)
	}
}

func TestSSHReportsTheDirectoryOfTheCopyItNames(t *testing.T) {
	m := agentMachine(sshMachine(), agentKey)
	s := sshStep(m, &effecttest.Out{})
	s.Conf.IdentityPubKey = sshcfg.AgentKeys(agentKey)[0].Pub()
	got := reportOf(t, s)
	for _, name := range []string{"/h/.config", "/h/.config/ws", "/h/" + sshcfg.KeyCopies} {
		if it := got[name]; it.Kind != step.DirItem || it.Found != "missing" {
			t.Errorf("%s = %+v, want a directory, missing", name, it)
		}
	}
}
