package tui

import (
	"bytes"
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRunLogsEveryLineClean(t *testing.T) {
	var log, out bytes.Buffer
	c := noKeys(&out)
	c.Log = &log
	err := Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		b.Say("a line")
		b.Warn("take care")
		_, _ = b.pipe.Write([]byte("\x1b[31mred\x1b[0m\n"))
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "a line\nwarning: take care\nred\n"
	if log.String() != want {
		t.Errorf("log %q, want %q", log.String(), want)
	}
}
