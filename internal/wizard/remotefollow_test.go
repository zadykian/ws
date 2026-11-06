package wizard

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
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

func newFollow(m effect.Machine) *Remote {
	return &Remote{SSH: SSH{M: m, Lock: "/t/l"}, Script: []byte("echo\n")}
}

func TestFollowHandsEventsOnAndPassesTheLog(t *testing.T) {
	m := effecttest.NewMachine()
	r := newFollow(m)
	r.Events = &heard{}
	m.Results[remoteLine(r, "run", "--only", "wireguard")] = effecttest.Result{Out: []byte(runLog)}
	if err := r.Follow(context.Background(), "--only", "wireguard"); err != nil {
		t.Fatal(err)
	}
	if got := r.Events.(*heard).steps(); !slices.Equal(got, []string{"ssh/keys"}) {
		t.Errorf("events = %q; want the line's one", got)
	}
	if want := []string{"ssh: changed", "last line"}; !slices.Equal(m.Passed, want) {
		t.Errorf("Passed = %q, want %q", m.Passed, want)
	}
}
