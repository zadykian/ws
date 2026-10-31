package daemon

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

// peerKey is ws's key in tunnel.conf, and macKey the Mac's own, each 32 equal bytes in base64.
const (
	peerKey = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
	macKey  = "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="
)

var errTest = errors.New("no")

var t0 = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

// fakeMachine answers each command from a table by its line, and with success where it has none. A
// mv moves a file.
type fakeMachine struct {
	*effecttest.Machine
	mu    sync.Mutex
	runs  []string
	table map[string]effecttest.Result
}

func (m *fakeMachine) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	line := strings.Join(append([]string{name}, args...), " ")
	m.runs = append(m.runs, line)
	if name == "mv" {
		m.Files[args[2]] = m.Files[args[1]]
		delete(m.Files, args[1])
	}
	r := m.table[line]
	return r.Out, r.Err
}

// fakeDevice keeps what IpcSet takes and answers IpcGet with get.
type fakeDevice struct {
	sets []string
	get  string
}

func (d *fakeDevice) IpcGet() (string, error) { return d.get, nil }

func (d *fakeDevice) IpcSet(text string) error {
	d.sets = append(d.sets, text)
	return nil
}

// rig is a Daemon on fakes, with a clock the test moves and no timer that fires on its own.
type rig struct {
	*Daemon
	m   *fakeMachine
	dev *fakeDevice
	log *bytes.Buffer
	now time.Time
}

func newRig(conf string) *rig {
	m := &fakeMachine{Machine: effecttest.NewMachine(), table: map[string]effecttest.Result{}}
	m.Files[dir+"/private.key"] = []byte(macKey + "\n")
	m.Files[dir+"/tunnel.conf"] = []byte("WS_PUBLIC_KEY=" + peerKey + "\n" + conf)
	r := &rig{m: m, dev: &fakeDevice{}, log: &bytes.Buffer{}, now: t0}
	r.Daemon = New(m, r.dev, "utun4", dir, "/var/run/ws-tunnel.state", r.log)
	r.Now = func() time.Time { return r.now }
	return r
}

// play sends the events, then Stop, and runs the loop to its end.
func (r *rig) play(events ...Event) error {
	for _, e := range append(events, Stop) {
		r.Send(e)
	}
	return r.Run(context.Background())
}

// lines returns the log without its times.
func (r *rig) lines() []string {
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(r.log.String()), "\n") {
		if l != "" {
			out = append(out, l[len("2006-01-02 15:04:05 "):])
		}
	}
	return out
}
