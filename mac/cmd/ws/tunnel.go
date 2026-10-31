package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zadykian/ws/mac/internal/daemon"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/wgdev"
)

const tunnelUsage = "usage: ws tunnel run\n"

// stateFile is what the daemon writes for status, and removes where it stops.
const stateFile = "/var/run/ws-tunnel.state"

// watchWait is how long the watch of network changes waits before it starts notifyutil again.
const watchWait = 5 * time.Second

// tunnel runs ws tunnel's subcommands.
func tunnel(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "run" {
		return tunnelRun(stdout, stderr)
	}
	_, _ = io.WriteString(stderr, tunnelUsage)
	return 2
}

// tunnelRun keeps the tunnel up as root: the device on a utun, and the daemon's loop, which logs to
// stdout, launchd's log file. SIGHUP reads the settings again, and SIGTERM stops.
func tunnelRun(stdout, stderr io.Writer) int {
	// Subscribe before anything else: launchd gives each signal its default action, which ends the
	// daemon on a SIGHUP.
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	tn, err := wgdev.Open(stdout)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws: %v\n", err)
		return 1
	}
	defer tn.Close()
	d := daemon.New(effect.OSMachine{}, tn, tn.Name(), daemon.Dir, stateFile, stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Signals(ctx, sigs)
	go d.Watch(ctx, watchWait, daemon.Child(daemon.NetworkWatch...))
	go d.Exit(ctx, tn.Done())
	if err := d.Run(ctx); err != nil {
		return 1
	}
	return 0
}
