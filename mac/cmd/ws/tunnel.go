package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zadykian/ws/mac/internal/daemon"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/uapi"
	"github.com/zadykian/ws/mac/internal/wgdev"
)

const tunnelUsage = "usage: ws tunnel run | status | key\n"

// stateFile is what the daemon writes for status, and removes where it stops.
const stateFile = "/var/run/ws-tunnel.state"

// logFile is launchd's file for the daemon's output, which the daemon starts anew past a size.
const logFile = "/var/log/ws-tunnel.log"

// watchWait is how long the watch of network changes waits before it starts notifyutil again.
const watchWait = 5 * time.Second

// statusWait is how long status waits for the device's socket.
const statusWait = 5 * time.Second

// tunnel runs ws tunnel's subcommands.
func tunnel(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "run" {
		return tunnelRun(stdout, stderr)
	}
	if len(args) == 1 && args[0] == "status" {
		return tunnelStatus(os.Geteuid(), stdout, stderr)
	}
	if len(args) == 1 && args[0] == "key" {
		return tunnelKey(os.Geteuid(), stdout, stderr)
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
	d.Log = logFile
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

// tunnelStatus prints the way in use and the peer's handshake and bytes. The state file and the
// device's socket are root's, so it needs root, as ws-tunnel status did.
func tunnelStatus(uid int, stdout, stderr io.Writer) int {
	if uid != 0 {
		_, _ = io.WriteString(stderr, "ws tunnel status: run it as root, with sudo\n")
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), statusWait)
	defer cancel()
	text, err := daemon.Status(ctx, effect.OSMachine{}, uapi.Get, daemon.Dir, stateFile, time.Now())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws: %v\n", err)
		return 1
	}
	_, _ = io.WriteString(stdout, text)
	return 0
}

// tunnelKey makes the Mac's private key where none is, and prints the public key. The private key
// stays in root's file, which only root can write, so setup runs this through sudo.
func tunnelKey(uid int, stdout, stderr io.Writer) int {
	if uid != 0 {
		_, _ = io.WriteString(stderr, "ws tunnel key: run it as root, with sudo\n")
		return 1
	}
	public, err := daemon.Key(context.Background(), effect.OSMachine{}, rand.Reader, daemon.Dir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, public)
	return 0
}
