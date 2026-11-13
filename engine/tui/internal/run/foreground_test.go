package run

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/step"
)

// TestForegroundHoldsTheScreenOffForTheCommand runs a command that cannot start. Its failure shows
// that the screen left first and came back after.
func TestForegroundHoldsTheScreenOffForTheCommand(t *testing.T) {
	var log []string
	b := newBridge(nil)
	b.suspend = func() error { log = append(log, "suspend"); return nil }
	b.resume = func() error { log = append(log, "resume"); return nil }
	err := b.Foreground(local.OSMachine{}).PassForeground(t.Context(), false, "ws-no-such-command")
	if err == nil {
		t.Error("a command that does not exist ran")
	}
	if want := []string{"suspend", "resume"}; !slices.Equal(log, want) {
		t.Errorf("log = %q, want %q", log, want)
	}
}

// TestForegroundPrintsDirectionsOnTheTerminalItLeft follows two commands of a step: the line before
// the first prints between the screen's leaving and the command, and the next has none to repeat.
func TestForegroundPrintsDirectionsOnTheTerminalItLeft(t *testing.T) {
	var log []string
	var printed bytes.Buffer
	b := newBridge(&printed)
	b.send = func(tea.Msg) {}
	b.suspend = func() error { log = append(log, "suspend, "+printed.String()); return nil }
	b.resume = func() error { log = append(log, "resume, "+printed.String()); return nil }
	b.Say("ssh-keygen asks its passphrase twice")
	for range 2 {
		_ = b.Foreground(local.OSMachine{}).PassForeground(t.Context(), false, "ws-no-such-command")
	}
	said := "ssh-keygen asks its passphrase twice\n"
	if want := []string{"suspend, ", "resume, " + said, "suspend, " + said, "resume, " + said}; !slices.Equal(log, want) {
		t.Errorf("log = %q, want %q", log, want)
	}
}

// TestForegroundHasNoDirectionsOfAnEarlierStep starts a step past a line of the one before it.
func TestForegroundHasNoDirectionsOfAnEarlierStep(t *testing.T) {
	var printed bytes.Buffer
	b := newBridge(&printed)
	b.send = func(tea.Msg) {}
	b.pipe = &pipe{bridge: b}
	b.Say("the step before is done")
	b.Observe(step.Event{Step: "keys", Kind: step.Started})
	_ = b.Foreground(local.OSMachine{}).PassForeground(t.Context(), false, "ws-no-such-command")
	if printed.Len() != 0 {
		t.Errorf("printed %q, want nothing", printed.String())
	}
}

// TestForegroundTakesTheTicketWhileTheScreenCanAsk runs as root a command that cannot start, past a
// Sudo that names it. A second look at the ticket would come after the suspension.
func TestForegroundTakesTheTicketWhileTheScreenCanAsk(t *testing.T) {
	var log []string
	b := newBridge(nil)
	b.suspend = func() error { log = append(log, "suspend"); return nil }
	b.resume = func() error { log = append(log, "resume"); return nil }
	m := local.OSMachine{Sudo: []string{"ws-no-such-sudo"}, Ticket: func() error { log = append(log, "ticket"); return nil }}
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
	m := local.OSMachine{Ticket: func() error { return refused }}
	if err := b.Foreground(m).PassForeground(t.Context(), true, "true"); !errors.Is(err, refused) {
		t.Errorf("err = %v, want the ticket's", err)
	}
}

func TestForegroundTakesNoTicketForTheUsersCommand(t *testing.T) {
	m := local.OSMachine{Ticket: func() error { return errors.New("a ticket for the user's command") }}
	if err := newBridge(nil).Foreground(m).PassForeground(t.Context(), false, "true"); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestForegroundRunsNothingWhereTheScreenCannotLeave(t *testing.T) {
	boom := errors.New("boom")
	b := newBridge(nil)
	b.suspend = func() error { return boom }
	b.resume = func() error { t.Error("resumed a screen that never left"); return nil }
	err := b.Foreground(local.OSMachine{}).PassForeground(t.Context(), false, "true")
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want the suspension's", err)
	}
}

func TestForegroundReportsAScreenThatCannotResume(t *testing.T) {
	boom := errors.New("boom")
	b := newBridge(nil)
	b.resume = func() error { return boom }
	if err := b.Foreground(local.OSMachine{}).PassForeground(t.Context(), false, "true"); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the resume's", err)
	}
}
