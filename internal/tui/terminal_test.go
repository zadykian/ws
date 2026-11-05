package tui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/effect"
)

// endsOnTheNormalScreen fails where the screen came back past its end.
func endsOnTheNormalScreen(t *testing.T, out string) {
	t.Helper()
	if strings.LastIndex(out, "\x1b[?1049h") > strings.LastIndex(out, leaveBuffer) {
		t.Errorf("the screen came back past its end: %q", out)
	}
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
