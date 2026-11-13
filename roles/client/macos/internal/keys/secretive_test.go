package keys_test

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/resource/file"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

func TestSecretiveKeys(t *testing.T) {
	ctx := context.Background()
	m := effecttest.NewMachine()
	if keys, socket, err := keys.SecretiveKeys(ctx, m, mactest.AgentHome); keys != nil || socket || err != nil {
		t.Errorf("no socket: %v, %v, %v", keys, socket, err)
	}
	mactest.AgentMachine(m, mactest.AgentKey, mactest.AgentGit)
	if keys, socket, err := keys.SecretiveKeys(ctx, m, mactest.AgentHome); len(keys) != 2 || !socket || err != nil {
		t.Errorf("two keys: %v, %v, %v", keys, socket, err)
	}
	// ssh-add's words for no keys, with its exit 1.
	m.Results[mactest.AgentList] = effecttest.Result{Out: []byte("The agent has no identities.\n"), Err: errors.New("exit status 1")}
	if keys, socket, err := keys.SecretiveKeys(ctx, m, mactest.AgentHome); len(keys) != 0 || !socket || err != nil {
		t.Errorf("an empty agent: %v, %v, %v", keys, socket, err)
	}
	// The socket stays after Secretive quits.
	m.Results[mactest.AgentList] = effecttest.Result{Err: errors.New("exit status 2: Connection refused")}
	if _, socket, err := keys.SecretiveKeys(ctx, m, mactest.AgentHome); !socket || !errors.Is(err, keys.ErrAgentDown) {
		t.Errorf("a stale socket: %v, %v", socket, err)
	}
}

// macOS 27 denies ws every file in Secretive's container: ReadKey must never need one.
func TestReadKeyTakesTheAgentsKeyWithNoFile(t *testing.T) {
	m := mactest.AgentMachine(effecttest.NewMachine(), mactest.AgentKey)
	pub := sshcfg.AgentKeys(mactest.AgentKey)[0].Pub()
	id, err := keys.ReadKey(context.Background(), m, conf.Conf{IdentityPubKey: pub}, mactest.AgentHome, keys.LoginKey)
	if err != nil {
		t.Fatal(err)
	}
	if id.Pub != pub || id.File != mactest.CopyOf(mactest.AgentKey) || id.Text != mactest.AgentKey ||
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
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	pub := mactest.CopyOf(mactest.AgentKey)
	for name, args := range map[string][2]string{"a .pub elsewhere": {"/h/.ssh/k.pub", mactest.AgentKey}, "no text": {pub, ""}} {
		if got, err := keys.PlanCopy(m, out, args[0], args[1], mactest.AgentHome); got != nil || err != nil {
			t.Errorf("%s: %v, %v", name, got, err)
		}
	}
	got, err := keys.PlanCopy(m, out, pub, mactest.AgentKey, mactest.AgentHome)
	if err != nil || len(got) != 4 || got[2] != (file.UserMkdir{M: m, Path: "/h/" + sshcfg.KeyCopies, Mode: 0o755}) ||
		got[3] != (keys.Copy{M: m, Out: out, Path: pub, Text: mactest.AgentKey}) {
		t.Fatalf("a missing copy: %v, %v; want the directories, then the copy", got, err)
	}
	m.Results["mkdir -p /h/"+sshcfg.KeyCopies] = effecttest.Result{}
	for _, c := range got {
		if err := change.Apply(context.Background(), m, &effecttest.Out{}, c); err != nil {
			t.Fatal(err)
		}
	}
	if string(m.Files[pub]) != mactest.AgentKey {
		t.Errorf("the copy holds %q", m.Files[pub])
	}
	if got, err := keys.PlanCopy(m, out, pub, mactest.AgentKey, mactest.AgentHome); got != nil || err != nil {
		t.Errorf("a copy that holds the key: %v, %v", got, err)
	}
}
