package tui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// leaveBuffer is what puts a terminal back on its normal screen.
const leaveBuffer = "\x1b[?1049l"

// noKeys is a terminal that nobody types on.
func noKeys(out io.Writer) Config {
	return Config{In: strings.NewReader(""), Out: out, Theme: testTheme, Cut: testCut}
}

func TestRunGivesTheScreenTheLinesAndReturnsTheError(t *testing.T) {
	failed := errors.New("a step failed")
	var got []tea.Msg
	var out bytes.Buffer
	err := Run(t.Context(), noKeys(&out), recorder{&got}, func(_ context.Context, b *Bridge) error {
		b.Say("a line")
		b.ByHand("restart ws")
		return failed
	})
	if !errors.Is(err, failed) {
		t.Errorf("Run: %v", err)
	}
	if !slices.Contains(got, tea.Msg(Line{Said, "a line"})) {
		t.Errorf("the screen got %v", got)
	}
	left := strings.LastIndex(out.String(), leaveBuffer)
	if left < 0 || !strings.Contains(out.String()[left:], "restart ws") {
		t.Errorf("no line by hand past the screen's end in %q", out.String())
	}
}
