package daemon

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/uapi"
)

// Run sets up the utun, then keeps the tunnel up until Stop, or until ctx ends. It logs why it
// ends with an error. The caller made the device and its utun, and closes them, which takes the
// address and the routes with them.
func (d *Daemon) Run(ctx context.Context) error {
	if err := d.start(ctx); err != nil {
		d.say("%v", err)
		return err
	}
	defer d.stop()
	d.say("started on %s, %s", d.Tun, tunnel.Address)
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-d.Events:
			switch e {
			case Stop:
				return nil
			case Exited:
				d.say("wireguard-go has exited")
				return errors.New("wireguard-go has exited")
			}
		}
	}
}

// start reads the settings and gives the device its key, the utun its address, and ws's address a
// route into the utun.
func (d *Daemon) start(ctx context.Context) error {
	var err error
	if d.set, err = ReadSettings(ctx, d.M, d.Dir); err != nil {
		return err
	}
	for _, w := range d.set.Warnings {
		d.say("%s", w)
	}
	if d.key, err = ReadKey(d.M, d.Dir); err != nil {
		return err
	}
	address := tunnel.Address.String()
	text, err := uapi.SetPrivateKey(d.key)
	if err == nil {
		err = d.Dev.IpcSet(text)
	}
	if err == nil {
		err = d.run(ctx, "ifconfig", d.Tun, "inet", address+"/32", address, "alias")
	}
	if err == nil {
		err = d.run(ctx, "ifconfig", d.Tun, "up")
	}
	if err == nil {
		err = d.run(ctx, "route", "-q", "-n", "add", "-inet", tunnel.Server.String()+"/32", "-interface", d.Tun)
	}
	if err != nil {
		return fmt.Errorf("cannot set up %s: %w", d.Tun, err)
	}
	return nil
}

// stop removes the state file, which says that the daemon runs.
func (d *Daemon) stop() {
	_ = d.run(context.Background(), "rm", "-f", d.StateFile, d.StateFile+".new")
	d.say("stopped")
}
