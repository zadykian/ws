package keys_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/macos/internal/keys"
	"github.com/zadykian/ws/macos/internal/mactest"
	"github.com/zadykian/ws/macos/values"
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
	id, err := keys.ReadKey(context.Background(), m, values.Conf{IdentityPubKey: pub}, mactest.AgentHome, keys.LoginKey)
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

func TestCopiedLeavesOutAPathThatLeadsOut(t *testing.T) {
	out := sshcfg.KeyCopies + "/../../../Users/u/k.pub"
	if keys.Copied(out, mactest.AgentKey) {
		t.Errorf("Copied(%q) = true", out)
	}
	if got := keys.CopyParts(effecttest.NewMachine(), &effecttest.Out{}, out, mactest.AgentKey); len(got) != 0 {
		t.Errorf("CopyParts(%q) = %v, want none", out, got)
	}
}

func TestPlanCopy(t *testing.T) {
	ctx := context.Background()
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	pub := mactest.CopyOf(mactest.AgentKey)
	for name, args := range map[string][2]string{"a .pub elsewhere": {"/h/.ssh/k.pub", mactest.AgentKey}, "no text": {pub, ""}} {
		if got, err := keys.PlanCopy(ctx, m, out, args[0], args[1]); got != nil || err != nil {
			t.Errorf("%s: %v, %v", name, got, err)
		}
	}
	got, err := keys.PlanCopy(ctx, m, out, pub, mactest.AgentKey)
	if err != nil || len(got) != 3 || !strings.HasPrefix(got[0].String(), "create directory /etc/ws,") ||
		!strings.HasPrefix(got[1].String(), "create directory "+sshcfg.KeyCopies+",") ||
		!strings.HasPrefix(got[2].String(), "create "+pub+"\n") {
		t.Fatalf("a missing copy: %v, %v; want /etc/ws, the directory of the copies, then the copy", got, err)
	}
	if err := (resource.Group{}).Apply(ctx, step.Plan{Changes: got}); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[pub]) != mactest.AgentKey {
		t.Errorf("the copy holds %q", m.Files[pub])
	}
	m.Calls = nil
	if got, err := keys.PlanCopy(ctx, m, out, pub, mactest.AgentKey); len(got) != 0 || err != nil {
		t.Errorf("a copy that holds the key: %v, %v", got, err)
	}
	if i := slices.IndexFunc(m.Calls, func(c string) bool { return strings.Contains(c, "root ") }); i >= 0 {
		t.Errorf("the second plan asked for root: %q", m.Calls[i])
	}
}
