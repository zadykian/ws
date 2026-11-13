package settings

import (
	"context"
	"fmt"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/uapi"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// GetFunc reads the device's state from the UAPI socket at sock, as uapi.Get does.
type GetFunc func(ctx context.Context, sock string) (string, error)

// Status returns what ws client macos tunnel status prints. That is the state that the daemon wrote
// at p.State, and the peer's handshake and bytes, which the device behind its tun's socket
// reports. The peer is the ws key that p.Conf holds. A device that does not answer
// leaves the handshake as none and the transfer out, as the state file alone is still true.
func Status(ctx context.Context, m effect.Machine, get GetFunc, p Paths, now time.Time) (string, error) {
	text, err := m.ReadFile(p.State)
	if err != nil {
		return "", fmt.Errorf("ws client macos tunnel isn't running; its log is %s", p.Log)
	}
	state, err := tunnel.ParseState(string(text))
	if err != nil {
		return "", err
	}
	conf, err := m.ReadFile(p.Conf)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: run ws client macos apply", p.Conf)
	}
	c, err := tunnel.ParseConf(string(conf))
	if err != nil {
		return "", err
	}
	var peer tunnel.Stats
	if got, err := get(ctx, uapi.Socket(state.Interface)); err == nil {
		if info, err := uapi.ParseGet(got, c.Key); err == nil {
			peer = info.Stats
		}
	}
	return state.Format(now, peer), nil
}
