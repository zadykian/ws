package run

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// textRun runs a work that writes "the key" to its Effects.Text, and returns what Config.Text
// took and what the log holds.
func textRun(t *testing.T, c Config, fail error) (stdout, log string, err error) {
	var out, logged bytes.Buffer
	c.Text, c.Log = &out, &logged
	err = Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		_, _ = io.WriteString(b.Effects().Text, "the key\n")
		return fail
	})
	return out.String(), logged.String(), err
}

func TestRunWritesTheTextPastTheScreen(t *testing.T) {
	c := noKeys(io.Discard)
	stdout, log, err := textRun(t, c, nil)
	if err != nil || stdout != "the key\n" || !strings.Contains(log, "the key") {
		t.Errorf("err = %v, stdout = %q, log = %q", err, stdout, log)
	}
}

// A failed run logs its text, as a plain run printed it, and writes none.
func TestRunLogsTheTextOfAFailedRunAndWritesNone(t *testing.T) {
	stdout, log, err := textRun(t, noKeys(io.Discard), errors.New("boom"))
	if err == nil || stdout != "" || !strings.Contains(log, "the key") {
		t.Errorf("err = %v, stdout = %q, log = %q", err, stdout, log)
	}
}

func TestRunDropsTheTextWithNoWriter(t *testing.T) {
	c := noKeys(io.Discard)
	err := Run(t.Context(), c, recorder{new([]tea.Msg)}, func(_ context.Context, b *Bridge) error {
		_, werr := io.WriteString(b.Effects().Text, "the key\n")
		return werr
	})
	if err != nil {
		t.Errorf("Run: %v", err)
	}
}

// A pick screen follows a run that says the same again, so it writes nothing.
func TestRunQuietWritesNoText(t *testing.T) {
	c := noKeys(io.Discard)
	c.quiet = true
	if stdout, _, err := textRun(t, c, nil); err != nil || stdout != "" {
		t.Errorf("err = %v, stdout = %q", err, stdout)
	}
}

func TestOffscreenDiscardsTheText(t *testing.T) {
	e := Offscreen(nil).Effects()
	if n, err := io.WriteString(e.Text, "x"); n != 1 || err != nil {
		t.Errorf("wrote %d, %v", n, err)
	}
	if (*Bridge)(nil).Effects().Text != nil {
		t.Error("a nil Bridge has a text")
	}
}
