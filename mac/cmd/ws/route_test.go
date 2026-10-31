package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestTunnelLoadedFollowsLaunchd(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{Err: errors.New("not loaded")}
	if got := tunnelLoaded(context.Background(), nil, m, &bytes.Buffer{}); got != 1 {
		t.Errorf("not loaded: status = %d; want 1", got)
	}
	m.Results["launchctl print system/com.github.zadykian.ws-tunnel"] = effecttest.Result{}
	if got := tunnelLoaded(context.Background(), nil, m, &bytes.Buffer{}); got != 0 {
		t.Errorf("loaded: status = %d; want 0", got)
	}
}
