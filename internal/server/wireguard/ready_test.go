package wireguard

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
)

func newReady(m *effecttest.Machine, out *effecttest.Out, slept *int) ready {
	return ready{m: m, out: out, tries: 3, pause: time.Second, sleep: func(time.Duration) { *slept++ }}
}

func TestReadyShowsTheKeyAndWaitsForWg0(t *testing.T) {
	m, public := machine("OTHER")
	out, slept := &effecttest.Out{}, 0
	cs, err := newReady(m, out, &slept).Plan(t.Context())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v; want the wait", cs, err)
	}
	if got := strings.Join(out.Lines, "\n"); !strings.Contains(got, "for WS_PUBLIC_KEY in the Mac's mac.conf: "+public) {
		t.Errorf("output = %q", got)
	}
	if err := cs[0].Apply(t.Context()); err == nil || !strings.Contains(err.Error(), public) || slept != 2 {
		t.Errorf("Apply = %v after %d pauses, want a refusal after 2", err, slept)
	}
}

func TestReadyIsDoneWhereWg0HoldsTheKey(t *testing.T) {
	m, public := machine("")
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public + "\n")}
	if cs, err := newReady(m, &effecttest.Out{}, new(int)).Plan(t.Context()); err != nil || len(cs) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", cs, err)
	}
}

func TestReadyWaitsThroughAMissingWg0(t *testing.T) {
	m, public := machine("")
	m.Results["wg show wg0 public-key"] = effecttest.Result{Err: errors.New("no such device")}
	slept := 0
	cs, err := newReady(m, &effecttest.Out{}, &slept).Plan(t.Context())
	if err != nil || len(cs) != 1 {
		t.Fatalf("Plan = %v, %v", cs, err)
	}
	m.Results["wg show wg0 public-key"] = effecttest.Result{Out: []byte(public)}
	if err := cs[0].Apply(t.Context()); err != nil || slept != 0 {
		t.Errorf("Apply = %v after %d pauses; want done at once", err, slept)
	}
}

func TestReadySaysThatADryRunMakesNoKey(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/run/systemd/system"] = 0o755
	if cs, err := newReady(m, out, new(int)).Plan(t.Context()); err != nil || len(cs) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", cs, err)
	}
	if got := strings.Join(out.Lines, "\n"); !strings.Contains(got, "has no WireGuard key yet") {
		t.Errorf("output = %q", got)
	}
}

func TestReadyChecksNothingWithoutSystemd(t *testing.T) {
	m, public := machine("OTHER")
	delete(m.Dirs, "/run/systemd/system")
	out := &effecttest.Out{}
	if cs, err := newReady(m, out, new(int)).Plan(t.Context()); err != nil || len(cs) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", cs, err)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), public) {
		t.Errorf("output = %q, want the key", out.Lines)
	}
}
