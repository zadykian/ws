package run

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/zadykian/ws/engine/core/step"
)

func TestRunLogsEveryLineClean(t *testing.T) {
	var log, out bytes.Buffer
	c := noKeys(&out)
	c.Log, c.LogPath = &log, "/state/ws/run.log"
	err := Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		b.Say("a line")
		_, _ = b.pipe.Write([]byte("\x1b[31mred\x1b[0m\n"))
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "a line\nred\n"
	if log.String() != want {
		t.Errorf("log %q, want %q", log.String(), want)
	}
	if strings.Contains(out.String(), "/state/ws/run.log") {
		t.Errorf("a clean run names its log: %q", out.String())
	}
}

// pastScreen gives what a reader sees once a run has left the screen. Bubble Tea's queries of the
// terminal land there too, unless the environment names an ssh session.
func pastScreen(out string) string {
	_, past, _ := strings.Cut(out, leaveBuffer)
	return ansi.Strip(past)
}

// A step that warns passes, so the line is all that shows what it gave up.
func TestRunPrintsAWarningPastTheScreen(t *testing.T) {
	var log, out bytes.Buffer
	c := noKeys(&out)
	c.Log, c.LogPath = &log, "/state/ws/run.log"
	err := Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		b.Warn("take care")
		b.Warn("take care")
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := "warning: take care\n"; log.String() != want {
		t.Errorf("log %q, want %q", log.String(), want)
	}
	want := "warning: take care\nThe whole output is in /state/ws/run.log\n"
	if got := pastScreen(out.String()); got != want {
		t.Errorf("past the screen %q, want %q, in %q", got, want, out.String())
	}
}

func TestRunReportsAFailedStepsOutputAndTheLog(t *testing.T) {
	var out bytes.Buffer
	c := noKeys(&out)
	c.LogPath = "/state/ws/run.log"
	broke := errors.New("exit 2")
	err := Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		b.Observe(step.Event{Kind: step.Started, Step: "docker"})
		b.Say("old output")
		b.Observe(step.Event{Kind: step.Done, Step: "docker"})
		b.Observe(step.Event{Kind: step.Started, Step: "tmux", Parent: "base"})
		b.Say("pulling")
		b.Observe(step.Event{Kind: step.Failed, Step: "tmux", Parent: "base", Err: broke})
		return broke
	})
	if !errors.Is(err, broke) {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	left := got[strings.LastIndex(got, leaveBuffer):]
	for _, want := range []string{"base: tmux failed", "  pulling\n", "is in /state/ws/run.log"} {
		if !strings.Contains(left, want) {
			t.Errorf("past the screen, no %q in %q", want, left)
		}
	}
	if strings.Contains(left, "old output") {
		t.Errorf("an earlier step's output in %q", left)
	}
}

// A failed Plan sends no Started, and --check none at all.
func TestRunReportsAPlanFailureFromItsOwnOutput(t *testing.T) {
	tests := []struct {
		name, plan, want string
	}{
		{"its Plan printed", "reading\n", "stage: b failed. The end of its output:\n  reading\n"},
		{"its Plan printed nothing", "", "stage: b failed.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			broke := errors.New("no file")
			err := Run(t.Context(), noKeys(&out), recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
				b.Say("output of a")
				b.Observe(step.Event{Kind: step.Done, Step: "a", Parent: "stage", Check: true})
				_, _ = b.pipe.Write([]byte(tt.plan))
				b.Observe(step.Event{Kind: step.Failed, Step: "b", Parent: "stage", Check: true, Err: broke})
				b.Observe(step.Event{Kind: step.Skipped, Step: "c", Parent: "stage", Check: true})
				b.Observe(step.Event{Kind: step.Failed, Step: "stage", Check: true, Err: broke})
				return broke
			})
			if !errors.Is(err, broke) {
				t.Fatalf("Run: %v", err)
			}
			if got := pastScreen(out.String()); got != tt.want {
				t.Errorf("past the screen %q, want %q, in %q", got, tt.want, out.String())
			}
		})
	}
}

func TestRunReportsCheckChanges(t *testing.T) {
	var out bytes.Buffer
	c := noKeys(&out)
	c.LogPath = "/state/ws/run.log"
	err := Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		b.Observe(step.Event{Kind: step.Planned, Check: true, Step: "ssh", Parent: "base",
			Changes: []step.Change{oneChange("write /etc/motd")}})
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := out.String()
	left := got[strings.LastIndex(got, leaveBuffer):]
	if !strings.Contains(left, "base: ssh: write /etc/motd\n") || !strings.Contains(left, "/state/ws/run.log") {
		t.Errorf("past the screen: %q", left)
	}
}

// A plain run prints its stdout text and what stays by hand, so the log holds them. The window
// shows neither. A line SayLasting keeps by hand is logged once by its Line alone.
func TestRunLogsWhatAPlainRunPrints(t *testing.T) {
	var log, out bytes.Buffer
	c := noKeys(&out)
	c.Log = &log
	err := Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		b.ByHand("%s: run it again", "ws")
		b.SayLasting("%s", "set the key")
		b.SayLasting("%s", "set the key")
		b.Log("\x1b[31mpublic key\x1b[0m\nthen\n")
		b.Log("")
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "ws: run it again\nset the key\nset the key\npublic key\nthen\n"
	if log.String() != want {
		t.Errorf("log %q, want %q", log.String(), want)
	}
}
