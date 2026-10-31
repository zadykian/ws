package daemon

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/uapi"
)

// LogFile is launchd's log of the daemon, which status names where the daemon is not running.
const LogFile = "/var/log/ws-tunnel.log"

// GetFunc reads the device's state from the UAPI socket at sock, as uapi.Get does.
type GetFunc func(ctx context.Context, sock string) (string, error)

// Status returns what ws tunnel status prints: the state that the daemon wrote at stateFile, and
// the peer's handshake and bytes, which the device behind its tun's socket reports. The peer is
// ws's key in dir's tunnel.conf. A device that does not answer leaves the handshake as none and
// the transfer out, as the state file alone is still true.
func Status(ctx context.Context, m effect.Machine, get GetFunc, dir, stateFile string, now time.Time) (string, error) {
	text, err := m.ReadFile(stateFile)
	if err != nil {
		return "", fmt.Errorf("ws tunnel isn't running; its log is %s", LogFile)
	}
	state, err := tunnel.ParseState(string(text))
	if err != nil {
		return "", err
	}
	conf, err := m.ReadFile(path.Join(dir, "tunnel.conf"))
	if err != nil {
		return "", fmt.Errorf("cannot read %s/tunnel.conf: run mac/setup.sh", dir)
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
