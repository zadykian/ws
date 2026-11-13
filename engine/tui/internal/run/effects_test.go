package run

import (
	"context"
	"io"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// effectsOf returns what Bridge.Effects gives a work under a screen of m.
func effectsOf(t *testing.T, m effect.Machine) Effects {
	c := noKeys(io.Discard)
	c.Machine = m
	var e Effects
	_ = Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error { e = b.Effects(); return nil })
	return e
}

// The screen's machine keeps its commands off the terminal and asks for sudo's password, and Talks
// suspends the screen for that machine.
func TestEffectsWrapTheScreensMachine(t *testing.T) {
	e := effectsOf(t, effect.OSMachine{Confine: "/home/z"})
	m, ok := e.M.(effect.OSMachine)
	if !ok || m.Stdin != effect.NoStdin || m.Ticket == nil || m.Confine != "/home/z" {
		t.Fatalf("machine = %+v", e.M)
	}
	if f, ok := e.Talks.(foreground); !ok || f.m.Ticket == nil {
		t.Errorf("talks = %#v, want the screen's foreground on the wrapped machine", e.Talks)
	}
}

// A fake has no terminal to keep its commands off: it stands unchanged, and so does its own
// foreground. A test's nil Bridge has no effects.
func TestEffectsKeepAFake(t *testing.T) {
	fake := effecttest.NewMachine()
	e := effectsOf(t, fake)
	if e.M != effect.Machine(fake) {
		t.Errorf("a fake became %T", e.M)
	}
	if _, ok := effect.Machine(fake).(effect.Foreground); ok != (e.Talks != nil) {
		t.Errorf("talks = %#v", e.Talks)
	}
	if e := (*Bridge)(nil).Effects(); e.M != nil || e.Talks != nil {
		t.Errorf("a nil Bridge gave %+v", e)
	}
	if e := Offscreen(fake).Effects(); e.M != effect.Machine(fake) {
		t.Errorf("Offscreen gave %+v", e)
	}
}

// A work gives its steps the screen's observer and stop by Options, and a nil Bridge's options
// stand as they were.
func TestOptionsGiveTheBridgeAndItsStop(t *testing.T) {
	b := Offscreen(effecttest.NewMachine())
	o := b.Options(step.Options{Check: true, Parent: "p"})
	if o.Observer != step.Observer(b) || !o.Check || o.Parent != "p" {
		t.Errorf("options = %+v", o)
	}
	if o.Stop == nil || o.Stop() != nil {
		t.Error("Stop is nil or fails, want one that returns nil")
	}
	if !b.took.Load() {
		t.Error("Options left the run as one that took none")
	}
	if got := (*Bridge)(nil).Options(step.Options{Check: true}); got.Observer != nil || !got.Check {
		t.Errorf("a nil Bridge gave %+v", got)
	}
}
