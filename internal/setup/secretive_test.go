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
