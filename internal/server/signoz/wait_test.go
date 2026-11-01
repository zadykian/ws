package signoz

import (
	"context"
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
