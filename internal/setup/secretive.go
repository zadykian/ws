package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/sshcfg"
)

// ErrAgentDown is why Secretive's keys cannot be listed: its socket stays after Secretive quits, and
// nothing answers there.
var ErrAgentDown = errors.New("Secretive's agent does not answer") //nolint:staticcheck // Secretive is a name

// SecretiveKeys returns the keys that Secretive's agent lists, and whether its socket is there. It
// asks the agent, as macOS 27 denies ws the files of Secretive's container (docs/mac.md). A socket
// with no agent is ErrAgentDown.
func SecretiveKeys(ctx context.Context, m effect.Machine, home string) ([]sshcfg.AgentKey, bool, error) {
	socket := home + "/" + sshcfg.SecretiveData + "/socket.ssh"
	info, err := m.Stat(socket)
	if err != nil || info.Mode&fs.ModeSocket == 0 {
		return nil, false, nil
	}
	out, err := m.Run(ctx, "env", "SSH_AUTH_SOCK="+socket, "ssh-add", "-L")
	// ssh-add exits 1 and says so on its standard output for an agent with no keys.
	if err != nil && !strings.Contains(string(out), "The agent has no identities") {
		return nil, true, fmt.Errorf("%w at %s: open Secretive, then run this again, or name another key in "+
			"mac.conf's WS_IDENTITY", ErrAgentDown, socket)
	}
	return sshcfg.AgentKeys(string(out)), true, nil
}
