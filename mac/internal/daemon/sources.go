package daemon

import (
	"bufio"
	"context"
	"io"
	"os"
	"syscall"
	"time"
)

// Signals turns the signals of sigs into events until ctx ends: SIGHUP reads the settings again,
// and SIGTERM and SIGINT stop. launchd starts the daemon with each signal's default action, under
// which a SIGHUP would end it, so the caller subscribes before the loop runs, and a SIGHUP in
// between waits in sigs.
func (d *Daemon) Signals(ctx context.Context, sigs <-chan os.Signal) {
	for {
		select {
		case <-ctx.Done():
			return
		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				d.Send(Reload)
			case syscall.SIGTERM, syscall.SIGINT:
				d.Send(Stop)
			}
		}
	}
}

// Watch sends Changed for each line that start's reader gives, which is notifyutil's: a line each
// time configd posts a network change, as when an interface's IPv4 address comes or goes. Where
// the reader ends, it starts again after wait, until ctx ends.
func (d *Daemon) Watch(ctx context.Context, wait time.Duration, start func(context.Context) (io.ReadCloser, error)) {
	for {
		r, err := start(ctx)
		if err != nil {
			d.say("cannot watch network changes: %v", err)
		} else {
			lines := bufio.NewScanner(r)
			for lines.Scan() {
				d.Send(Changed)
			}
			_ = r.Close()
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}
