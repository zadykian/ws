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

// TestForegroundTakesTheTicketWhileTheScreenCanAsk runs as root a command that cannot start, past a
// Sudo that names it. A second look at the ticket would come after the suspension.
func TestForegroundTakesTheTicketWhileTheScreenCanAsk(t *testing.T) {
	var log []string
	b := newBridge(nil)
	b.suspend = func() error { log = append(log, "suspend"); return nil }
	b.resume = func() error { log = append(log, "resume"); return nil }
	m := effect.OSMachine{Sudo: []string{"ws-no-such-sudo"}, Ticket: func() error { log = append(log, "ticket"); return nil }}
	if err := b.Foreground(m).PassForeground(t.Context(), true, "true"); err == nil {
		t.Error("a sudo that does not exist ran")
	}
	if want := []string{"ticket", "suspend", "resume"}; !slices.Equal(log, want) {
		t.Errorf("log = %q, want %q", log, want)
	}
}

func TestForegroundKeepsTheScreenWhereSudoRefuses(t *testing.T) {
	refused := errors.New("refused")
	b := newBridge(nil)
	b.suspend = func() error { t.Error("the screen left for a command with no ticket"); return nil }
	m := effect.OSMachine{Ticket: func() error { return refused }}
	if err := b.Foreground(m).PassForeground(t.Context(), true, "true"); !errors.Is(err, refused) {
		t.Errorf("err = %v, want the ticket's", err)
	}
}

func TestForegroundTakesNoTicketForTheUsersCommand(t *testing.T) {
	m := effect.OSMachine{Ticket: func() error { return errors.New("a ticket for the user's command") }}
	if err := newBridge(nil).Foreground(m).PassForeground(t.Context(), false, "true"); err != nil {
		t.Errorf("err = %v", err)
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
