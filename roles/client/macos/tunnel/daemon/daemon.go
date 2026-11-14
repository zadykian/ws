// Package daemon runs ws client macos tunnel's loop. It reads the Mac and the peer, asks the pure
// functions of package tunnel what to do, and does it through effect.Machine and a Device.
package daemon

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/netip"
	"os"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/settings"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// Device is the wireguard-go device, as the daemon sees it: its UAPI.
type Device interface {
	IpcGet() (string, error)
	IpcSet(string) error
}

// Daemon keeps the Mac's tunnel to ws up. A loop in Run takes its events from one channel, and
// each event reads the Mac and the peer, asks package tunnel what to do, and does it. Only the loop
// touches the fields below Events, so they need no lock.
type Daemon struct {
	M   effect.Machine
	Dev Device
	Tun string // the utun's name
	// Paths are the daemon's files: State is what status reads, and Log is the file that trimLog
	// starts anew past LogLimit, empty for none.
	Paths settings.Paths
	Out   io.Writer // the log
	Now   func() time.Time
	// CheckEvery is how often a round looks for the handshake. TickEvery is how long a tick comes
	// after the last tick ended. ChangeWait is how long the loop waits after a network change.
	// LookupWait bounds the lookups of the other servers' remotes, which hold up the loop.
	CheckEvery, TickEvery, ChangeWait, LookupWait time.Duration
	// Program returns the file of the program that runs, root's copy of ws. Nil means the
	// executable's file.
	Program func() (fs.FileInfo, error)
	// Events is the channel of the loop. Signals, timers and the watch of network changes send to
	// it through Send.
	Events chan Event

	program    fs.FileInfo // the program's file as start found it, nil where it could not
	set        settings.Settings
	key        string
	inUse      tunnel.Candidate // Tier TierNone where no path answers; empty before the first round
	since      time.Time        // when inUse began
	peerConf   tunnel.Conf      // the settings that the last round used
	seen       []tunnel.Iface   // the Mac's addresses at the start of the last round
	resolved   netip.Addr       // ws's name as the last round resolved it
	misses     int              // pings of ws missed in a row
	round      []tunnel.Attempt // the endpoints tried since the round began, empty between rounds
	allowed    tunnel.Allowed   // what the last try left out
	ends       []netip.Addr     // the endpoints of the last round, which the routes stay clear of
	elsewhere  []netip.Addr     // the other servers' endpoints, as the last round found them
	routed     []netip.Prefix
	unroutable []netip.Prefix
	told       []netip.Prefix
}

// Event is what the loop reacts to.
type Event int

// The events.
const (
	Tick    Event = iota // TickEvery after the last tick ended
	Check                // look again at the choice, as a round waits for a handshake
	Reload               // SIGHUP
	Changed              // configd posted a network change
	Settled              // ChangeWait after a change
	Stop                 // SIGTERM
	Exited               // wireguard-go has exited
)

// New returns a Daemon with an Events channel, for the files at p.
func New(m effect.Machine, dev Device, tun string, p settings.Paths, out io.Writer) *Daemon {
	return &Daemon{
		M: m, Dev: dev, Tun: tun, Paths: p, Out: out, Now: time.Now,
		CheckEvery: time.Second, TickEvery: 10 * time.Second, ChangeWait: 2 * time.Second, LookupWait: 2 * time.Second,
		Events: make(chan Event, 64),
	}
}

// Send puts an event in the loop's channel.
func (d *Daemon) Send(e Event) { d.Events <- e }

// say logs a line, with the time.
func (d *Daemon) say(format string, a ...any) {
	_, _ = fmt.Fprintf(d.Out, "%s %s\n", d.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, a...))
}

// programFile returns the file of the program that runs.
func (d *Daemon) programFile() (fs.FileInfo, error) {
	if d.Program != nil {
		return d.Program()
	}
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return os.Stat(path)
}

func (d *Daemon) later(e Event, after time.Duration) {
	time.AfterFunc(after, func() { d.Send(e) })
}

func (d *Daemon) run(ctx context.Context, name string, args ...string) error {
	_, err := d.M.Run(ctx, name, args...)
	return err
}
