package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	p256    = "ecdsa-sha2-nistp256 AAAA me\n"
	gitP256 = "ecdsa-sha2-nistp256 BBBB me\n"
	mldsa   = "ssh-mldsa65 CCCC me\n"
	reason  = "a P-256 key for ws"
)

func TestMacKeyPickWaits(t *testing.T) {
	k, m, out := newMacKey(t, "")
	agent(m, gitP256)
	naps := 0
	k.Sleep = func(d time.Duration) {
		if naps++; naps == 3 && d == 2*time.Second {
			agent(m, gitP256, p256)
		}
	}
	got, err := k.pick(context.Background(), copyOf(gitP256), reason, "")
	if err != nil || got != copyOf(p256) || naps != 3 || len(out.Lines) != 2 {
		t.Errorf("pick = %q, %v after %d naps, saying %q", got, err, naps, out.Lines)
	}
	if string(m.Files[got]) != p256 {
		t.Errorf("the copy holds %q", m.Files[got])
	}
}

// The socket stays after Secretive quits: the wizard opens it, and its agent starts during the wait.
func TestMacKeyPickWaitsForAnAgentThatStarts(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m)
	m.Results[listKeys] = effecttest.Result{Err: errors.New("exit status 2: Connection refused")}
	k.Sleep = func(time.Duration) { agent(m, p256) }
	if got, err := k.pick(context.Background(), "", reason, ""); err != nil || got != copyOf(p256) {
		t.Errorf("pick = %q, %v", got, err)
	}
}

func TestMacKeyPickStopsWithTheContext(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m)
	ctx, cancel := context.WithCancel(context.Background())
	naps := 0
	k.Sleep = func(time.Duration) {
		if naps++; naps == 2 {
			cancel()
		}
	}
	if _, err := k.pick(ctx, "", reason, ""); !errors.Is(err, context.Canceled) || naps != 2 {
		t.Errorf("pick after a signal = %v after %d naps", err, naps)
	}
}

func TestMacKeyPickRefuses(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m, mldsa)
	if _, err := k.pick(context.Background(), "", reason, ""); err == nil || !strings.Contains(err.Error(), "is not the P-256 key") {
		t.Errorf("pick of an ML-DSA key = %v", err)
	}
	agent(m, mldsa, p256)
	if _, err := k.pick(context.Background(), "", reason, ""); err == nil || !strings.Contains(err.Error(), "holds several keys") {
		t.Errorf("pick of two keys = %v", err)
	}
}
