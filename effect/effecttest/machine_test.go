package effecttest_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

var (
	_ effect.Foreground = (*effecttest.Machine)(nil)
	_ effect.Lines      = (*effecttest.Machine)(nil)
)

func TestMachinePassForeground(t *testing.T) {
	boom := errors.New("boom")
	m := effecttest.NewMachine()
	m.Results["ssh-keygen -q"] = effecttest.Result{Err: boom}
	m.Results["installer -pkg"] = effecttest.Result{}
	ctx := context.Background()
	if err := m.PassForeground(ctx, false, "ssh-keygen", "-q"); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the result's", err)
	}
	if err := m.PassForeground(ctx, true, "installer", "-pkg"); err != nil {
		t.Errorf("as root: %v", err)
	}
	want := []string{"foreground ssh-keygen -q", "foreground root installer -pkg"}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("Calls = %q, want %q", m.Calls, want)
	}
}

func TestMachinePassLines(t *testing.T) {
	boom := errors.New("boom")
	m := effecttest.NewMachine()
	m.Results["ssh ws"] = effecttest.Result{Out: []byte("event\nlog\n\nlast"), Err: boom}
	take := func(line string) bool { return line == "event" }
	if err := m.PassLines(context.Background(), []byte("in"), take, "ssh", "ws"); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the result's", err)
	}
	if want := []string{"log", "", "last"}; !slices.Equal(m.Passed, want) {
		t.Errorf("Passed = %q, want %q", m.Passed, want)
	}
	if !slices.Equal(m.Calls, []string{"run ssh ws"}) || !slices.Equal(m.Inputs, []string{"in"}) {
		t.Errorf("Calls = %q, Inputs = %q", m.Calls, m.Inputs)
	}
}
