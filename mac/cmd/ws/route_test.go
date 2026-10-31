package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
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

func TestTunnelRoutesCheckShowsTheFileAndWritesNothing(t *testing.T) {
	m := effecttest.NewMachine()
	m.Root["/etc/ws-tunnel/routes.d"] = effect.Attrs{Mode: 0o755}
	var stderr bytes.Buffer
	got := tunnelRoutes(context.Background(), []string{"--check", "test", "198.51.100.0/24"}, m, &stderr)
	if got != 0 || !strings.Contains(stderr.String(), "+198.51.100.0/24") {
		t.Errorf("status = %d, stderr = %q", got, stderr.String())
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "write") {
			t.Errorf("--check made %q", c)
		}
	}
}

func TestRouteCommandsPrintTheErrorAndExit1(t *testing.T) {
	var stderr bytes.Buffer
	if got := tunnelRoutes(context.Background(), []string{"test", "0.0.0.0/0"}, effecttest.NewMachine(), &stderr); got != 1 ||
		stderr.String() != "ws: ws tunnel routes test: 0.0.0.0/0 overlaps the tunnel's 10.99.0.0/24\n" {
		t.Errorf("routes: status = %d, stderr = %q", got, stderr.String())
	}
	stderr.Reset()
	if got := resolverCommand(context.Background(), nil, effecttest.NewMachine(), &stderr); got != 2 ||
		!strings.HasPrefix(stderr.String(), "usage: ws resolver") {
		t.Errorf("resolver: status = %d, stderr = %q", got, stderr.String())
	}
}
