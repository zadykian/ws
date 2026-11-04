package wizard

import (
	"context"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/daemon"
	"github.com/zadykian/ws/internal/tunnel"
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
	// Conf is mac.conf's path, which the direct way to ws comes from.
	Conf string
}

// First returns the address that the Host key stage asked, or an empty string.
func (s SSH) First() string {
	b, err := s.M.ReadFile(s.Lock + "/" + FirstFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// way finds the direct way to ws from mac.conf and the Mac's addresses, with the address that the
// Host key stage asked in place of the way's host. A failed ifconfig counts as no addresses, as in
// the daemon. A mac.conf that can't be read or parsed leaves no way. With no way and that address,
// the port stays ssh's own.
func (s SSH) way(ctx context.Context) (tunnel.Direct, bool) {
	var d tunnel.Direct
	var found bool
	if text, err := s.M.ReadFile(s.Conf); err == nil {
		if c, errs := conf.Parse(string(text)); len(errs) == 0 {
			out, err := s.M.Run(ctx, "ifconfig")
			if err != nil {
				out = nil
			}
			d, found = tunnel.DirectWay(daemon.ParseIfconfig(string(out), ""), tunnelConf(c), c.RemotePort)
		}
	}
	if first := s.First(); first != "" {
		d.Host, found = first, true
	}
	return d, found
}

// tunnelConf takes the keys of mac.conf that a direct way reads. conf checked them already.
func tunnelConf(c conf.Conf) tunnel.Conf {
	t := tunnel.Conf{Remote: c.Remote}
	t.LAN, _ = netip.ParsePrefix(c.LAN)
	t.LANServer, _ = netip.ParseAddr(c.LANAddress)
	return t
}

// room tells whether the lock's path leaves room for a socket's. A master binds its socket at the
// path plus a dot and 16 characters, then links it into place (mux.c's muxserver_listen), and
// macOS's sun_path holds 103 bytes. So a path may hold 86, and the lock, which a slash and
// a name of 8 characters, direct.1, follow, 77. A Mac's holds 63. ssh also reads % and $ in a path
// as its own.
func (s SSH) room() bool {
	return !strings.ContainsAny(s.Lock, "%$") && len(s.Lock) <= 77
}

// shared returns the options that every call needs, with the master's socket in the lock. ssh stops
// where known_hosts lacks ws's key, or holds another, and adds nothing there. The first call opens
// a master connection for the calls after it, so a key that asks Touch ID at each login, as
// Secretive's does, asks once a run.
//   - A call that uses a master takes the master's options, so those that every caller needs are
//     here. BatchMode stays with the calls, as only a login reads it.
//   - A master lives 30 minutes past its last call, which outlasts a Homebrew install between two.
//   - It runs no LocalCommand, as config.d/ws's that starts cld's watcher in a Ghostty pane.
func (s SSH) shared(socket string) []string {
	var a []string
	if s.room() {
		a = append(a, "-o", "ControlMaster=auto", "-o", "ControlPath="+s.Lock+"/"+socket,
			"-o", "ControlPersist=30m")
	}
	return append(a, "-o", "ConnectTimeout=10", "-o", "StrictHostKeyChecking=yes", "-o", "UpdateHostKeys=no",
		"-o", "CheckHostIP=no", "-o", "ServerAliveInterval=15", "-o", "PermitLocalCommand=no")
}

// argv returns ssh's arguments for a direct call, whose host in args is ws.
//   - The way's host and port go before args as options, over config.d/ws's, which leads through the
//     tunnel. The address that the Host key stage asked replaces the way's host.
//   - AddressFamily=inet goes on every call: the ways are IPv4 by conf and WS_REMOTE takes IPv4 alone.
//   - The socket is direct, so that a call never rides the tunnel's master. The address that the
//     Host key stage asked adds .1, so that a master made before it never carries a call to it.
func (s SSH) argv(ctx context.Context, args []string) []string {
	d, found := s.way(ctx)
	socket := "direct"
	if s.First() != "" {
		socket += ".1"
	}
	a := s.shared(socket)
	if found {
		a = append(a, "-o", "HostName="+d.Host)
		if d.Port != "" {
			a = append(a, "-o", "Port="+d.Port)
		}
	}
	return append(append(a, "-o", "AddressFamily=inet"), args...)
}

// Run runs ssh with the arguments, and returns its standard output.
func (s SSH) Run(ctx context.Context, args ...string) ([]byte, error) {
	return s.M.Run(ctx, "ssh", s.argv(ctx, args)...)
}

// Tunnel runs ssh through the tunnel, as config.d/ws leads, on a master of its own in the lock and
// with none of the way's options.
func (s SSH) Tunnel(ctx context.Context, args ...string) ([]byte, error) {
	return s.M.Run(ctx, "ssh", append(s.shared("%n"), args...)...)
}
