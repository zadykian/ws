package daemon

import "os"

// LogLimit is the size, 1 MiB, past which the log starts anew.
const LogLimit = 1 << 20

// trimLog starts the log anew past LogLimit, at the daemon's start and at each tick. launchd opened
// the file to append, so the next line goes at its new end.
func (d *Daemon) trimLog() {
	if d.Log == "" {
		return
	}
	if fi, err := os.Stat(d.Log); err != nil || fi.Size() <= LogLimit {
		return
	}
	if err := os.Truncate(d.Log, 0); err != nil {
		d.say("cannot start the log anew: %v", err)
		return
	}
	d.say("started the log anew past 1 MiB")
}
