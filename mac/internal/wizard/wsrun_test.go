package wizard

import (
	"context"
	"testing"
)

func TestRunDone(t *testing.T) {
	r := &WsRun{Remote: &Remote{status: map[string]string{
		"run": "abc 0 off", "active": "-", "outbound": "off"}, read: true}}
	ctx := context.Background()
	if !r.runDone(ctx, "cloud") || !r.runDone(ctx, "lan") {
		t.Error("runDone is false after a run that exited 0")
	}
	r.Update = true
	if r.runDone(ctx, "lan") {
		t.Error("runDone is true under --update before the stage ran")
	}
	r.Update, r.Remote.status["run"] = false, "abc 1 on"
	if r.runDone(ctx, "lan") {
		t.Error("runDone is true after a run that exited 1")
	}
}
