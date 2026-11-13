package wizard

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// runLog is a run's log, as the job's journal gives it: ws's lines, with an event among them.
const runLog = "ssh: changed\n{\"event\":\"done\",\"parent\":\"ssh\",\"step\":\"keys\",\"index\":3}\nlast line\n"

// heard is the Events of a test.
type heard []step.Event

func (h *heard) Observe(e step.Event) { *h = append(*h, e) }

// steps returns the events' steps.
func (h heard) steps() []string {
	var names []string
	for _, e := range h {
		names = append(names, e.Parent+"/"+e.Step)
	}
	return names
}

// newFollow returns a session over ws's fake, whose runs' events go to events.
func newFollow(m *effecttest.Machine, events step.Observer) *session.Session {
	reach := sessiontest.Reach{Reach: host.MacReach{SSH: remote.SSH{M: m, Lock: "/t/l"}}, M: m}
	return session.New(reach, session.Scripts{Remote: []byte("echo\n"), Files: []byte("true\n"), Events: events})
}

func TestFollowHandsEventsOnAndPassesTheLog(t *testing.T) {
	m := effecttest.NewMachine()
	h := &heard{}
	r := newFollow(m, h)
	m.Results[sessiontest.Op("run", "--only", "wireguard")] = effecttest.Result{Out: []byte(runLog)}
	if err := r.Run(context.Background(), []string{"--only", "wireguard"}); err != nil {
		t.Fatal(err)
	}
	if got := h.steps(); !slices.Equal(got, []string{"ssh/keys"}) {
		t.Errorf("events = %q; want the line's one", got)
	}
	if want := []string{"ssh: changed", "last line"}; !slices.Equal(m.Passed, want) {
		t.Errorf("Passed = %q, want %q", m.Passed, want)
	}
}

func TestFollowHearsAReplayAgain(t *testing.T) {
	m := effecttest.NewMachine()
	h := &heard{}
	r := newFollow(m, h)
	m.Results[sessiontest.Op("run")] = effecttest.Result{Out: []byte(runLog)}
	for range 2 {
		if err := r.Run(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(*h) != 2 {
		t.Errorf("heard %d events; want the event of each follow", len(*h))
	}
}

func TestFollowDropsEventsThatNoOneHears(t *testing.T) {
	m := effecttest.NewMachine()
	r := newFollow(m, nil)
	m.Results[sessiontest.Op("run")] = effecttest.Result{Out: []byte(runLog)}
	if err := r.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(m.Passed, func(l string) bool { return strings.Contains(l, "event") }) {
		t.Errorf("Passed = %q; want no event", m.Passed)
	}
}

func TestFollowReadsStatusAgainAfterAFailure(t *testing.T) {
	m := effecttest.NewMachine()
	h := &heard{}
	r := newFollow(m, h)
	drop := errors.New("ssh dropped")
	m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(session.Status{})}
	m.Results[sessiontest.Op("run")] = effecttest.Result{Out: []byte(runLog), Err: drop}
	r.Status(context.Background())
	if err := r.Run(context.Background(), nil); !errors.Is(err, drop) {
		t.Fatalf("Follow = %v, want ssh's error", err)
	}
	if len(*h) != 1 {
		t.Errorf("heard %d events; want those before the drop", len(*h))
	}
	m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(session.Status{Active: "ws-wizard-1"})}
	if s, _ := r.Status(context.Background()); s.Active != "ws-wizard-1" {
		t.Errorf("active = %q; want the status read again", s.Active)
	}
}

func TestFollowRunsOnAMachineWithoutLines(t *testing.T) {
	m := effecttest.NewMachine()
	ssh := remote.SSH{M: struct{ effect.Machine }{m}, Lock: "/t/l"}
	r := session.New(host.MacReach{SSH: ssh}, session.Scripts{Remote: []byte("echo\n")})
	args := []string{"-o", "BatchMode=yes", "ws", "sh", "-s", "--", "run"}
	m.Results["ssh "+strings.Join(ssh.Argv(context.Background(), args), " ")] = effecttest.Result{}
	if err := r.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got := m.Calls[len(m.Calls)-1]; !strings.HasSuffix(got, "-- run") {
		t.Errorf("last call = %q; want the run", got)
	}
}

// TestEachRunIsFollowed has each caller of the run op hear its log's event. Its status stays peer
// none, so the tunnel stops past the run, which is what this test reads.
func TestEachRunIsFollowed(t *testing.T) {
	ctx, h := context.Background(), &heard{}
	log := effecttest.Result{Out: []byte(runLog)}

	r, m := newRun("lan", idleRun, true, false)
	r.Ws.Hear(h)
	m.Results[sessiontest.Op("run")] = log
	plan, err := r.Plan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Apply(ctx, plan); err != nil {
		t.Fatal(err)
	}

	c, m, _ := newCable(t, session.Status{}, cableLink, nil)
	c.Ws.Hear(h)
	m.Results[sessiontest.Op("link", "eth1")] = effecttest.Result{}
	m.Results[sessiontest.Op("run", "--only", "network,wireguard")] = log
	if err := c.name(ctx, "eth1"); err != nil {
		t.Fatal(err)
	}

	tun, m := newTunnel(keyA, session.Status{}, false)
	tun.Ws.Hear(h)
	m.Results["sudo -v"] = effecttest.Result{}
	m.Results[libexec+" client macos tunnel key"] = effecttest.Result{Out: []byte(keyA + "\n")}
	m.Results[sessiontest.Op("peer", keyA)] = effecttest.Result{}
	m.Results[sessiontest.Op("run", "--only", "wireguard")] = log
	if err := tun.Apply(ctx, step.Plan{}); err == nil || !strings.Contains(err.Error(), "lacks the Mac's key") {
		t.Errorf("Apply = %v; want the stop past the run", err)
	}

	if len(*h) != 3 {
		t.Errorf("heard %d events; want one of each run", len(*h))
	}
}
