package engine

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/tui"
)

func TestStateOffTheScreenPrintsTheReport(t *testing.T) {
	c, out, _ := applyCall(strings.NewReader(""))
	c.JSON = true
	err := State(t.Context(), c, Target{State: func(context.Context, Call) (Report, error) {
		return Report{Print: func(_ context.Context, c Call) error {
			_, err := c.Out.Write([]byte("json=" + map[bool]string{true: "yes", false: "no"}[c.JSON]))
			return err
		}}, nil
	}})
	if want := "json=yes"; err != nil || out.String() != want {
		t.Errorf("err = %v, stdout = %q, want %q", err, out.String(), want)
	}
}

func TestStateStopsWhereTheTargetPreparesNoReport(t *testing.T) {
	c, _, _ := applyCall(strings.NewReader(""))
	boom := errors.New("boom")
	err := State(t.Context(), c, Target{State: func(context.Context, Call) (Report, error) { return Report{}, boom }})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

// The server's report takes a machine that ws_home confines, apart from the call's: the state
// screen's read must act on it.
func TestStateReadsOnTheReportsMachineNotTheCalls(t *testing.T) {
	keys, press := io.Pipe()
	t.Cleanup(func() { _ = press.Close() })
	c, _, _ := applyCall(keys)
	c.TTY, c.ErrTTY, c.OutTTY = true, true, true
	confined := effecttest.NewMachine()
	var read effect.Machine
	err := State(t.Context(), c, Target{State: func(context.Context, Call) (Report, error) {
		return Report{M: confined, Read: func(_ context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
			read = b.Effects().M
			_, _ = io.WriteString(press, "q")
			return nil, nil
		}}, nil
	}})
	if err != nil || read != confined {
		t.Errorf("err = %v, the read acted on %v, want the report's machine", err, read)
	}
}
