package main

import (
	"context"
	"io"

	"github.com/zadykian/ws/mac/internal/effect"
)

const (
	loadedUsage = "usage: ws tunnel loaded\n"
	daemonLabel = "com.github.zadykian.ws-tunnel"
)

// tunnelLoaded succeeds where launchd has the tunnel's daemon loaded. Anyone may read launchd's
// system domain, so it needs no sudo.
func tunnelLoaded(ctx context.Context, args []string, m effect.Machine, stderr io.Writer) int {
	if len(args) != 0 {
		_, _ = io.WriteString(stderr, loadedUsage)
		return 2
	}
	if _, err := m.Run(ctx, "launchctl", "print", "system/"+daemonLabel); err != nil {
		return 1
	}
	return 0
}
