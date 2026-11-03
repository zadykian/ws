package signoz

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func countingWait(tries int, slept *int) Wait {
	return Wait{Tries: tries, Delay: time.Second, Sleep: func(context.Context, time.Duration) error {
		*slept++
		return nil
	}}
}

func TestPollStopsWhenDone(t *testing.T) {
	var slept, calls int
	err := countingWait(5, &slept).poll(t.Context(), "x", func() (bool, string, error) {
		calls++
		return calls == 3, "not yet", nil
	})
	if err != nil || calls != 3 || slept != 2 {
		t.Errorf("err %v, %d calls, %d sleeps; want none, 3, 2", err, calls, slept)
	}
}

func TestPollGivesUpWithTheLastReason(t *testing.T) {
	var slept int
	err := countingWait(3, &slept).poll(t.Context(), "readyz", func() (bool, string, error) {
		return false, "status 503", nil
	})
	if err == nil || !strings.Contains(err.Error(), "readyz: gave up after 3 tries, last: status 503") {
		t.Errorf("err = %v, want a give-up with the last reason", err)
	}
	if slept != 2 {
		t.Errorf("slept %d times, want 2", slept)
	}
}

func TestPollReturnsAnErrorAtOnce(t *testing.T) {
	var slept int
	boom := errors.New("boom")
	err := countingWait(3, &slept).poll(t.Context(), "x", func() (bool, string, error) { return false, "", boom })
	if !errors.Is(err, boom) || slept != 0 {
		t.Errorf("err %v after %d sleeps, want boom and none", err, slept)
	}
}
