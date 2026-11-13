package tui

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

// quit is what the loop ends on.
var quit tea.Msg = tea.QuitMsg{}

// move sends h a move, and returns what the loop goes on with and the answer.
func move(h *handover, away bool) (tea.Msg, error) {
	reply := make(chan error, 1)
	next := h.filter(nil, terminalMsg{away, reply})
	return next, <-reply
}

// endsOnTheNormalScreen fails where the screen came back past its end.
func endsOnTheNormalScreen(t *testing.T, out string) {
	t.Helper()
	if strings.LastIndex(out, "\x1b[?1049h") > strings.LastIndex(out, leaveBuffer) {
		t.Errorf("the screen came back past its end: %q", out)
	}
}

func TestHandoverKeepsAQuitUntilTheTerminalIsBack(t *testing.T) {
	var log []string
	h := &handover{
		release: func() error { log = append(log, "release"); return nil },
		restore: func() error { log = append(log, "restore"); return nil },
	}
	if next, err := move(h, true); next != nil || err != nil {
		t.Fatalf("release: %v, %v", next, err)
	}
	if next := h.filter(nil, quit); next != nil {
		t.Errorf("a quit passed while a command had the terminal: %v", next)
	}
	if next, err := move(h, false); next != quit || err != nil {
		t.Errorf("restore: %v, %v", next, err)
	}
	if want := []string{"release", "restore"}; !slices.Equal(log, want) {
		t.Errorf("moves = %v, want %v", log, want)
	}
}

func TestHandoverPassesAQuitPastAFailedRelease(t *testing.T) {
	boom := errors.New("no terminal")
	h := &handover{release: func() error { return boom }}
	if _, err := move(h, true); !errors.Is(err, boom) || h.filter(nil, quit) != quit {
		t.Errorf("err = %v, away = %v", err, h.away)
	}
}

// TestRunEndsOnTheNormalScreenPastACommand gives the second signal while a command has the terminal.
func TestRunEndsOnTheNormalScreenPastACommand(t *testing.T) {
	var out bytes.Buffer
	err := Run(t.Context(), noKeys(&out), recorder{&[]tea.Msg{}}, func(ctx context.Context, b *Bridge) error {
		if err := b.suspend(); err != nil {
			return err
		}
		b.send(signalMsg{syscall.SIGTERM})
		b.send(signalMsg{syscall.SIGTERM})
		<-ctx.Done()
		time.Sleep(100 * time.Millisecond) // tea.Quit is a command, and reaches the loop by itself.
		err := b.resume()
		time.Sleep(100 * time.Millisecond) // A screen that came back past its end would draw by now.
		return err
	})
	if !errors.Is(err, testCut(syscall.SIGTERM)) {
		t.Errorf("Run: %v", err)
	}
	endsOnTheNormalScreen(t, out.String())
}

func TestRunRefusesACommandPastTheScreen(t *testing.T) {
	var out bytes.Buffer
	c := noKeys(&out)
	c.In = strings.NewReader("\x03\x03")
	_ = Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(ctx context.Context, b *Bridge) error {
		<-b.gone
		err := b.Foreground(effect.OSMachine{}).PassForeground(ctx, false, "true")
		if !errors.Is(err, errGone) {
			t.Errorf("PassForeground: %v", err)
		}
		return err
	})
	endsOnTheNormalScreen(t, out.String())
}
