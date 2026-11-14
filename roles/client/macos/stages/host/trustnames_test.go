package host

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
)

func TestHostKeyCheckNamesTheServer(t *testing.T) {
	h, _ := labHost(nil)
	h.Check = true
	if _, err := h.Plan(context.Background()); err != nil || !strings.Contains(strings.Join(h.Out.(*effecttest.Out).Lines, "\n"),
		"lab at 10.1.1.2:22 offers the ed25519 key "+hostFP+"; a run would ask whether lab's console shows it") {
		t.Errorf("--check = %v, said %q", err, h.Out.(*effecttest.Out).Lines)
	}
}

func TestHostKeyTrustNamesTheServer(t *testing.T) {
	w := remote.Way{Host: "10.1.1.2", Port: "22"}
	h, _ := labHost(nil)
	h.Tty = false
	if err := h.trust(context.Background(), w, hostFP); err == nil || !strings.Contains(err.Error(), "compare it with lab's console") {
		t.Errorf("trust with no terminal = %v; want lab's console", err)
	}
	h, _ = labHost(map[string]string{"WS_HOST_KEY": "SHA256:x"})
	if err := h.trust(context.Background(), w, hostFP); err == nil || !strings.Contains(err.Error(), "the key that lab offers at 10.1.1.2:22: compare that with lab's console, and check lab's address") {
		t.Errorf("trust with another key = %v; want lab", err)
	}
}

func TestHostKeyFingerprintNamesTheServer(t *testing.T) {
	h, m := labHost(nil)
	m.Results["ssh-keygen -l -f /l/host.pub"] = effecttest.Result{}
	if _, err := h.fingerprint(context.Background(), hostLine); err == nil || !strings.Contains(err.Error(), "the key that lab offers") {
		t.Errorf("fingerprint = %v; want lab", err)
	}
}
