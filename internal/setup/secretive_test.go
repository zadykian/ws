package setup_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
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
