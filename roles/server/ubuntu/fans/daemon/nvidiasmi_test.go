package daemon

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCommandStreamsTheLinesOfTheChild(t *testing.T) {
	g := &GPUs{Retry: time.Hour}
	ctx, stop := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { g.Run(ctx, Command("sh", "-c", "echo 0, 45; echo 1, 50; sleep 60")); close(done) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if v, err := g.Temperature(1); err == nil {
			if v != 50 {
				t.Fatalf("GPU 1 = %v, want 50", v)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no reading of GPU 1")
		}
		time.Sleep(time.Millisecond)
	}
	stop()
	<-done
}

func TestCommandWithoutTheProgramIsAStartError(t *testing.T) {
	_, err := Command("ws-no-such-program")(t.Context())
	if err == nil || !strings.Contains(err.Error(), "ws-no-such-program") {
		t.Errorf("start = %v, want an error naming the program", err)
	}
}
