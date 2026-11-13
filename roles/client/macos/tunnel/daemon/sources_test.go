package daemon

import (
	"context"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSignalsTurnIntoEvents(t *testing.T) {
	r := newRig("")
	sigs := make(chan os.Signal, 4)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.Signals(ctx, sigs); close(done) }()
	for _, s := range []os.Signal{syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR1} {
		sigs <- s
	}
	want := []Event{Reload, Stop, Stop}
	for _, w := range want {
		if got := <-r.Events; got != w {
			t.Errorf("event = %v, want %v", got, w)
		}
	}
	cancel()
	<-done
	if len(r.Events) != 0 {
		t.Errorf("SIGUSR1 sent %d events, want none", len(r.Events))
	}
}

func TestWatchSendsALineAChangeAndStartsAgain(t *testing.T) {
	r := newRig("")
	ctx, cancel := context.WithCancel(context.Background())
	starts := 0
	start := func(context.Context) (io.ReadCloser, error) {
		starts++
		switch starts {
		case 1:
			return io.NopCloser(strings.NewReader("com.apple.system.config.network_change\n" +
				"com.apple.system.config.network_change\n")), nil
		case 2:
			return nil, errTest
		}
		cancel()
		return io.NopCloser(strings.NewReader("")), nil
	}
	r.Watch(ctx, time.Millisecond, start)
	if starts != 3 || len(r.Events) != 2 {
		t.Errorf("starts = %d, events = %d, want 3 and 2", starts, len(r.Events))
	}
	if got := r.lines(); len(got) != 1 || !strings.HasPrefix(got[0], "cannot watch network changes") {
		t.Errorf("log = %q", got)
	}
}
