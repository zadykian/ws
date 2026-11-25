package tunnel

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/values"
)

// RemoveGuest deletes the tunnel that the host ended for a guest that is gone: its interface's
// files, its key, its peers' keys, then the interface. They are named by the slot, so the next guest
// of the slot would take them, and the old guest's peers would reach it.
func RemoveGuest(ctx context.Context, m effect.Machine, t values.GuestTunnel) error {
	if t.Slot == 0 {
		return errors.New("slot 0 is wg0's, the host's own tunnel, which no guest holds")
	}
	files := networkdDir + "/60-" + t.Link()
	if _, err := m.RunRoot(ctx, "rm", "-f", "--", files+".netdev", files+".network", t.KeyFile()); err != nil {
		return err
	}
	if _, err := m.RunRoot(ctx, "rm", "-rf", "--", t.PeersDir()); err != nil {
		return err
	}
	// networkd keeps an interface whose .netdev is gone, as networkctl(1) says of reload. The files
	// go first, so that no reload between makes the interface again.
	_, err := m.Stat("/sys/class/net/" + t.Link())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = m.RunRoot(ctx, "ip", "link", "delete", t.Link())
	return err
}
