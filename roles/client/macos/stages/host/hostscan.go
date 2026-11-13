package host

import (
	"context"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// scanKey returns the ed25519 key that ws offers at the way's host and port, as "ssh-ed25519 KEY",
// within 10 s. It goes over IPv4 alone where ssh takes it alone. It returns an empty string where
// ws offers none.
func scanKey(ctx context.Context, m effect.Machine, w remote.Way) string {
	args := []string{"-T", "10", "-t", "ed25519", "-p", w.Port, w.Host}
	if w.Inet {
		args = append([]string{"-4"}, args...)
	}
	out, _ := m.Run(ctx, "ssh-keyscan", args...)
	for _, l := range strings.Split(string(out), "\n") {
		if f := strings.Fields(l); len(f) >= 3 && f[1] == "ssh-ed25519" {
			return f[1] + " " + f[2]
		}
	}
	return ""
}

// fingerprint returns the fingerprint of the key, which ssh-keygen reads from a file in the lock.
func (h *Key) fingerprint(ctx context.Context, line string) (string, error) {
	file := h.Lock + "/host.pub"
	if err := h.M.WriteFile(file, []byte(line+"\n"), 0o600); err != nil {
		return "", err
	}
	out, err := h.M.Run(ctx, "ssh-keygen", "-l", "-f", file)
	if f := strings.Fields(string(out)); err == nil && len(f) > 1 {
		return f[1], nil
	}
	return "", fmt.Errorf("%s: ssh-keygen cannot read the key that ws offers", stage.Host)
}

// offer returns the sentence that names the key that ws offers.
func offer(w remote.Way, fp string) string {
	return "ws at " + w.Host + ":" + w.Port + " offers the ed25519 key " + fp
}
