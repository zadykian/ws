package daemon

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/tunnel"
	"github.com/zadykian/ws/internal/uapi"
)

// LogFile is launchd's log of the daemon, which status names where the daemon is not running.
const LogFile = "/var/log/ws-tunnel.log"

// GetFunc reads the device's state from the UAPI socket at sock, as uapi.Get does.
type GetFunc func(ctx context.Context, sock string) (string, error)

// Status returns what ws mac tunnel status prints. That is the state that the daemon wrote at
// stateFile, and the peer's handshake and bytes, which the device behind its tun's socket reports.
// The peer is the ws key that dir's tunnel.conf holds. A device that does not answer leaves the
// handshake as none and the transfer out, as the state file alone is still true.
func Status(ctx context.Context, m effect.Machine, get GetFunc, dir, stateFile string, now time.Time) (string, error) {
	text, err := m.ReadFile(stateFile)
	if err != nil {
		return "", fmt.Errorf("ws mac tunnel isn't running; its log is %s", LogFile)
	}
	state, err := tunnel.ParseState(string(text))
	if err != nil {
		return "", err
	}
	conf, err := m.ReadFile(path.Join(dir, "tunnel.conf"))
	if err != nil {
		return "", fmt.Errorf("cannot read %s/tunnel.conf: run ws mac apply", dir)
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
