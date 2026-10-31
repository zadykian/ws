package daemon

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"time"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/tunnel"
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
	// Tun is the utun's name, StateFile the file that status reads, and Dir the settings' directory.
	Tun, StateFile, Dir string
	Out                 io.Writer // the log
	Now                 func() time.Time
	// CheckEvery is how often a round looks for the handshake.
	CheckEvery time.Duration
	// Events is the channel of the loop. Signals, timers and the watch of network changes send to
	// it through Send.
	Events chan Event

	set        Settings
	key        string
	inUse      tunnel.Candidate // Tier TierNone where no path answers; empty before the first round
	since      time.Time        // when inUse began
	peerConf   tunnel.Conf      // the settings that the last round used
	seen       []tunnel.Iface   // the Mac's addresses at the start of the last round
	resolved   netip.Addr       // ws's name as the last round resolved it
	misses     int              // pings of ws missed in a row
	round      []tunnel.Attempt // the endpoints tried since the round began, empty between rounds
	allowed    tunnel.Allowed   // what the last try left out
	routed     []netip.Prefix
	unroutable []netip.Prefix
	told       []netip.Prefix
}

// Event is what the loop reacts to.
type Event int

// The events.
const (
	Tick   Event = iota // TickEvery after the last tick or reload ended
	Check               // look again at the choice, as a round waits for a handshake
	Stop                // SIGTERM
	Exited              // wireguard-go has exited
)

// New returns a Daemon with an Events channel, for the settings in dir.
func New(m effect.Machine, dev Device, tun, dir, stateFile string, out io.Writer) *Daemon {
	return &Daemon{
		M: m, Dev: dev, Tun: tun, Dir: dir, StateFile: stateFile, Out: out, Now: time.Now,
		CheckEvery: time.Second, Events: make(chan Event, 64),
	}
}

// Send puts an event in the loop's channel.
func (d *Daemon) Send(e Event) { d.Events <- e }

// say logs a line, with the time.
func (d *Daemon) say(format string, a ...any) {
	_, _ = fmt.Fprintf(d.Out, "%s %s\n", d.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, a...))
}

func (d *Daemon) later(e Event, after time.Duration) {
	time.AfterFunc(after, func() { d.Send(e) })
}

func (d *Daemon) run(ctx context.Context, name string, args ...string) error {
	_, err := d.M.Run(ctx, name, args...)
	return err
}
