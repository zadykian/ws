package engine

import (
	"context"
	"errors"
	"testing"
)

func TestStatePrintsTheReport(t *testing.T) {
	c, out, _ := applyCall()
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
	c, _, _ := applyCall()
	boom := errors.New("boom")
	err := State(t.Context(), c, Target{State: func(context.Context, Call) (Report, error) { return Report{}, boom }})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}
