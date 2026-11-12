package fans

import (
	"context"
	"fmt"
	"time"
)

// Release hands every fan back to the board's own curve.
func (d *Daemon) Release() {
	for _, c := range d.controls {
		if err := c.release(); err != nil {
			d.Log(c.name + ": " + err.Error())
		}
	}
}

// Run ticks every interval until ctx ends, then hands every fan back. A tick that cannot go on
// ends it with the error, after the fans are back.
func (d *Daemon) Run(ctx context.Context) error {
	defer d.Release()
	d.Log(fmt.Sprintf("%d fans, every %v s", len(d.controls), d.cfg.Interval))
	interval := time.Duration(d.cfg.Interval * float64(time.Second))
	deadline := d.now()
	for {
		if err := d.Tick(); err != nil {
			return err
		}
		deadline = deadline.Add(interval)
		if now := d.now(); deadline.Before(now) {
			deadline = now
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(deadline.Sub(d.now())):
		}
	}
}
