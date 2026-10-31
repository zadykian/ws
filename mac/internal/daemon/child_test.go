package daemon

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestChildLines(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := New(nil, nil, "utun5", "/d", "/s", &bytes.Buffer{})
	go d.Watch(ctx, time.Hour, Child("printf", "a\nb\n"))
	for range 2 {
		select {
		case e := <-d.Events:
			if e != Changed {
				t.Fatalf("event = %v, want Changed", e)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("no event for a line")
		}
	}
}

func TestChildMissing(t *testing.T) {
	if _, err := Child("/nonexistent/ws-test")(context.Background()); err == nil {
		t.Error("a missing command started")
	}
}

func TestExit(t *testing.T) {
	d := New(nil, nil, "utun5", "/d", "/s", &bytes.Buffer{})
	done := make(chan struct{})
	close(done)
	go d.Exit(context.Background(), done)
	select {
	case e := <-d.Events:
		if e != Exited {
			t.Errorf("event = %v, want Exited", e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no Exited")
	}
}

func TestExitCancelled(t *testing.T) {
	d := New(nil, nil, "utun5", "/d", "/s", &bytes.Buffer{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d.Exit(ctx, make(chan struct{}))
	if len(d.Events) != 0 {
		t.Error("sent an event after ctx ended")
	}
}
