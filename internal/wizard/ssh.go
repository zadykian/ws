package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/effect"
)

// FirstFile is the file in the lock that holds the address the Host key stage asked, where ws did
// not answer at its way. Later stages read it as ssh's HostName, in place of a variable shared
// with that stage.
const FirstFile = "first"

// SSH runs ssh to ws-ssh for the stages, with the options every call needs.
type SSH struct {
	M effect.Machine
	// Lock is the run's lock, where the master connections' sockets live.
	Lock string
	// Home is the user's, which a ~/ in known_hosts' path means.
	Home string
}

// First returns the address that the Host key stage asked, or an empty string.
func (s SSH) First() string {
	b, err := s.M.ReadFile(s.Lock + "/" + FirstFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// room tells whether the lock's path leaves room for a socket's. A master binds its socket at the
// path plus a dot and 16 characters, then links it into place (mux.c's muxserver_listen), and
// macOS's sun_path holds 103 bytes. So a path may hold 86, and the lock, which a slash and
// ws-ssh.1 follow, 77. A Mac's holds 63. ssh also reads % and $ in a path as its own.
func (s SSH) room() bool {
	return !strings.ContainsAny(s.Lock, "%$") && len(s.Lock) <= 77
}

// argv returns ssh's arguments for a call. ssh stops where known_hosts lacks ws's key, or holds
// another, and adds nothing there. The first call opens a master connection in the lock for the
// calls after it, so a key that asks Touch ID at each login, as Secretive's does, asks once a run.
//   - A call that uses a master takes the master's options, so those that every caller needs are
//     here. BatchMode stays with the calls, as only a login reads it.
//   - A socket is named for the host that ssh is given. The address that the Host key stage asked
//     adds .1, so that a master made before it never carries a call to the new address.
//   - A master lives 30 minutes past its last call, which outlasts a Homebrew install between two.
//   - It runs no LocalCommand, as config.d/ws's that starts cld's watcher in a Ghostty pane.
func (s SSH) argv(_ context.Context, args []string) []string {
	first := s.First()
	suffix := ""
	if first != "" {
		suffix = ".1"
	}
	a := s.shared("%n" + suffix)
	if first != "" {
		a = append(a, "-o", "HostName="+first)
	}
	return append(a, args...)
}

// shared returns the options that every call needs, with the master's socket in the lock.
func (s SSH) shared(socket string) []string {
	var a []string
	if s.room() {
		a = append(a, "-o", "ControlMaster=auto", "-o", "ControlPath="+s.Lock+"/"+socket,
			"-o", "ControlPersist=30m")
	}
	return append(a, "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=yes", "-o", "UpdateHostKeys=no",
		"-o", "CheckHostIP=no", "-o", "ServerAliveInterval=15", "-o", "PermitLocalCommand=no")
}

// Tunnel runs ssh through the tunnel, as config.d/ws leads, on a master of its own in the lock, so
// that it never rides the master of a direct call.
func (s SSH) Tunnel(ctx context.Context, args ...string) ([]byte, error) {
	return s.M.Run(ctx, "ssh", append(s.shared("%n"), args...)...)
}

// Run runs ssh with the arguments, and returns its standard output.
func (s SSH) Run(ctx context.Context, args ...string) ([]byte, error) {
	return s.M.Run(ctx, "ssh", s.argv(ctx, args)...)
}
