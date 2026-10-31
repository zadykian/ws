package wizard

import (
	"context"
	"errors"
	"testing"
	"time"
)

const (
	keys   = "/h/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/PublicKeys"
	p256   = "ecdsa-sha2-nistp256 AAAA me\n"
	reason = "a P-256 key for ws"
)

func TestMacKeyPickWaits(t *testing.T) {
	k, m, out := newMacKey(t, "")
	m.Dirs[keys] = 0o755
	m.Files[keys+"/git.pub"], m.Files[keys+"/old-cert.pub"] = []byte(p256), []byte(p256)
	naps := 0
	k.Sleep = func(d time.Duration) {
		if naps++; naps == 3 && d == 2*time.Second {
			m.Files[keys+"/ws.pub"] = []byte(p256)
		}
	}
	got, err := k.pick(context.Background(), keys+"/git.pub", reason, "")
	if err != nil || got != keys+"/ws.pub" || naps != 3 || len(out.Lines) != 1 {
		t.Errorf("pick = %q, %v after %d naps, saying %q", got, err, naps, out.Lines)
	}
}

func TestMacKeyPickStopsWithTheContext(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	m.Dirs[keys] = 0o755
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
