package tui

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect"
)

// TestForegroundHoldsTheScreenOffForTheCommand runs a command that cannot start. Its failure shows
// that the screen left first and came back after.
func TestForegroundHoldsTheScreenOffForTheCommand(t *testing.T) {
	var log []string
	b := newBridge(nil)
	b.suspend = func() error { log = append(log, "suspend"); return nil }
	b.resume = func() error { log = append(log, "resume"); return nil }
	err := b.Foreground(effect.OSMachine{}).PassForeground(t.Context(), false, "ws-no-such-command")
	if err == nil {
		t.Error("a command that does not exist ran")
	}
	if want := []string{"suspend", "resume"}; !slices.Equal(log, want) {
		t.Errorf("log = %q, want %q", log, want)
	}
}

func TestForegroundRunsNothingWhereTheScreenCannotLeave(t *testing.T) {
	boom := errors.New("boom")
	b := newBridge(nil)
	b.suspend = func() error { return boom }
	b.resume = func() error { t.Error("resumed a screen that never left"); return nil }
	err := b.Foreground(effect.OSMachine{}).PassForeground(t.Context(), false, "true")
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want the suspension's", err)
	}
}

func TestForegroundReportsAScreenThatCannotResume(t *testing.T) {
	boom := errors.New("boom")
	b := newBridge(nil)
	b.resume = func() error { return boom }
	if err := b.Foreground(effect.OSMachine{}).PassForeground(t.Context(), false, "true"); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the resume's", err)
	}
}
