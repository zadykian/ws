package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/sshcfg"
)

const (
	p256    = "ecdsa-sha2-nistp256 AAAA me\n"
	gitP256 = "ecdsa-sha2-nistp256 BBBB me\n"
	mldsa   = "ssh-mldsa65 CCCC me\n"
	reason  = "a P-256 key for ws"
)

// listed is what the agent lists as the lines.
func listed(lines ...string) []sshcfg.AgentKey { return sshcfg.AgentKeys(strings.Join(lines, "")) }

func TestMacKeyWaits(t *testing.T) {
	k, m, out := newMacKey(t, "")
	agent(m, gitP256)
	naps := 0
	k.Sleep = func(d time.Duration) {
		if naps++; naps == 3 && d == 2*time.Second {
			agent(m, gitP256, p256)
		}
	}
	got, err := k.wait(context.Background(), listed(gitP256), reason)
	if err != nil || got.Line != strings.TrimSuffix(p256, "\n") || naps != 3 || len(out.Lines) != 1 {
		t.Errorf("wait = %q, %v after %d naps, saying %q", got.Line, err, naps, out.Lines)
	}
}

// The socket stays after Secretive quits: the wizard opens it, and its agent starts during the wait.
func TestMacKeyWaitsForAnAgentThatStarts(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m)
	m.Results[listKeys] = effecttest.Result{Err: errors.New("exit status 2: Connection refused")}
	k.Sleep = func(time.Duration) { agent(m, p256) }
	if got, err := k.wait(context.Background(), nil, reason); err != nil || got.Pub() != listed(p256)[0].Pub() {
		t.Errorf("wait = %q, %v", got.Line, err)
	}
}

func TestMacKeyWaitStopsWithTheContext(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m)
	ctx, cancel := context.WithCancel(context.Background())
	naps := 0
	k.Sleep = func(time.Duration) {
		if naps++; naps == 2 {
			cancel()
		}
	}
	if _, err := k.wait(ctx, nil, reason); !errors.Is(err, context.Canceled) || naps != 2 {
		t.Errorf("wait after a signal = %v after %d naps", err, naps)
	}
}

func TestMacKeyWaitRefusesAnotherType(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m, p256, mldsa)
	if _, err := k.wait(context.Background(), listed(p256), reason); err == nil ||
		!strings.Contains(err.Error(), "is not the P-256 key") {
		t.Errorf("wait for an ML-DSA key = %v", err)
	}
}
