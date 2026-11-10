package setup_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
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

// macOS 27 denies ws every file in Secretive's container: PickKey must never need one.
func TestPickKeyTakesTheAgentsKeyWithNoFile(t *testing.T) {
	m := agentMachine(effecttest.NewMachine(), agentKey)
	pub := sshcfg.AgentKeys(agentKey)[0].Pub()
	id, err := setup.PickKey(context.Background(), m, conf.Conf{IdentityPubKey: pub}, agentHome)
	if err != nil {
		t.Fatal(err)
	}
	want := sshcfg.Identity{Pub: pub, Agent: "~/" + sshcfg.SecretiveData + "/socket.ssh", Check: true, Text: agentKey}
	if id != want {
		t.Errorf("PickKey = %+v, want %+v", id, want)
	}
	for _, call := range m.Calls {
		if strings.Contains(call, "PublicKeys") {
			t.Errorf("PickKey touched Secretive's container: %s", call)
		}
	}
}

// ws mac apply takes no key that mac.conf leaves out, not even the agent's one. It names the line
// for each P-256 key, with its comment and fingerprint, and none for an ML-DSA key.
func TestPickKeyStopsWithoutANamedKey(t *testing.T) {
	const mldsa = "ssh-mldsa-65 AAAADHNzaC1tbGRzYS02NQ== ws\n"
	m := agentMachine(effecttest.NewMachine(), agentKey, mldsa, agentGit)
	_, err := setup.PickKey(context.Background(), m, conf.Conf{GitSigningPubKey: copyOf(agentGit)}, agentHome)
	var lines []string
	for _, line := range []string{agentKey, agentGit} {
		k := sshcfg.AgentKeys(line)[0]
		lines = append(lines, "WS_IDENTITY_PUB_KEY="+k.Pub()+" for "+k.Label())
	}
	want := "mac.conf names no WS_IDENTITY_PUB_KEY, ws's login key: run ws mac wizard, or add one of these lines " +
		"to mac.conf, " + strings.Join(lines, "; ")
	if err == nil || err.Error() != want {
		t.Errorf("PickKey = %v; want %q", err, want)
	}
	if _, err := setup.PickKey(context.Background(), effecttest.NewMachine(), conf.Conf{}, agentHome); err == nil ||
		!strings.Contains(err.Error(), "run ws mac wizard to pick or make one in Secretive") {
		t.Errorf("no agent: err = %v", err)
	}
}

// A mac.conf from before macOS 27 names the .pub in the container, which its copy replaces.
func TestPickKeyMovesAContainerKeyToItsCopy(t *testing.T) {
	name := sshcfg.AgentKeys(agentKey)[0].Name
	m := agentMachine(effecttest.NewMachine(), agentKey)
	c := conf.Conf{IdentityPubKey: "~/" + sshcfg.SecretiveData + "/PublicKeys/" + name}
	id, err := setup.PickKey(context.Background(), m, c, agentHome)
	if err != nil || sshcfg.Expand(id.Pub, agentHome) != copyOf(agentKey) || id.Text != agentKey {
		t.Errorf("PickKey = %+v, %v; want the copy", id, err)
	}
}

func TestPickKeyNamesACopyBeforeItIsThere(t *testing.T) {
	m := agentMachine(effecttest.NewMachine(), agentKey, agentGit)
	id, err := setup.PickKey(context.Background(), m, conf.Conf{IdentityPubKey: sshcfg.AgentKeys(agentGit)[0].Pub()}, agentHome)
	if err != nil || id.Text != agentGit {
		t.Errorf("PickKey = %+v, %v; want the key that the name gives", id, err)
	}
}

func TestPickKeyWithoutAnAgent(t *testing.T) {
	ctx := context.Background()
	m := agentMachine(effecttest.NewMachine())
	m.Results[agentList] = effecttest.Result{Err: errors.New("exit status 2: Connection refused")}
	if _, err := setup.PickKey(ctx, m, conf.Conf{}, agentHome); err == nil ||
		!strings.Contains(err.Error(), "mac.conf names no WS_IDENTITY_PUB_KEY") {
		t.Errorf("no WS_IDENTITY_PUB_KEY: err = %v", err)
	}
	// ws's copy serves while Secretive is closed.
	pub := copyOf(agentKey)
	m.Files[pub] = []byte(agentKey)
	m.Results["ssh-keygen -l -f "+pub] = effecttest.Result{}
	if id, err := setup.PickKey(ctx, m, conf.Conf{IdentityPubKey: pub}, agentHome); err != nil || id.Text != agentKey {
		t.Errorf("a copy: PickKey = %+v, %v", id, err)
	}
	// A .pub in the container does not, as macOS denies it.
	in := agentData + "/PublicKeys/a.pub"
	m.Root[in] = m.Root[agentSock]
	if _, err := setup.PickKey(ctx, m, conf.Conf{IdentityPubKey: in}, agentHome); err == nil ||
		!strings.Contains(err.Error(), "open Secretive") {
		t.Errorf("a .pub denied: err = %v", err)
	}
}

func TestPlanCopy(t *testing.T) {
	m := effecttest.NewMachine()
	pub := copyOf(agentKey)
	if got := setup.PlanCopy(m, "/h/.ssh/k.pub", agentKey, agentHome); got != nil {
		t.Errorf("a .pub elsewhere: %v", got)
	}
	if got := setup.PlanCopy(m, pub, "", agentHome); got != nil {
		t.Errorf("no text: %v", got)
	}
	got := setup.PlanCopy(m, pub, agentKey, agentHome)
	if len(got) != 1 || got[0] != (setup.KeyCopy{Path: pub, Text: agentKey}) {
		t.Fatalf("a missing copy: %v", got)
	}
	m.Results["mkdir -p /h/.config/ws/secretive"] = effecttest.Result{}
	if err := setup.Apply(context.Background(), m, &effecttest.Out{}, got[0]); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[pub]) != agentKey {
		t.Errorf("the copy holds %q", m.Files[pub])
	}
	if got := setup.PlanCopy(m, pub, agentKey, agentHome); got != nil {
		t.Errorf("a copy that holds the key: %v", got)
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
