package cable

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/cable/internal/netplan"
	"github.com/zadykian/ws/ubuntu/values"
)

// Check refuses what the cable's file must not act on: a port that is an uplink, and a networkd
// that does not run. file, where given, is the new netplan file. Check merges
// it with netplan's other files in a scratch root, and refuses it where netplan would not set up
// the port with it alone. The file is read as root, and no message quotes it. With no port, there is no
// cable, and nothing to refuse.
func Check(ctx context.Context, m effect.Machine, u effect.Units, c values.Cable, file string) error {
	if c.Port == "" {
		return nil
	}
	if _, err := m.Stat("/sys/class/net/" + c.Port); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("this machine has no interface %s: WS_CABLE_PORT names the port of the Mac's cable, as ip link lists it", c.Port)
	} else if err != nil {
		return err
	}
	port, err := readPort(ctx, m, c.Port, c.Address)
	if err != nil {
		return err
	}
	if file != "" {
		text, err := m.ReadRootFile(file)
		if err != nil {
			return err
		}
		problems, err := netplan.Check(ctx, m, text, port)
		if err != nil {
			return err
		}
		if len(problems) > 0 {
			return errors.New(strings.Join(problems, " "))
		}
	}
	return networkdRuns(ctx, u)
}

// networkdRuns fails where networkd, which netplan's file is for, does not run.
func networkdRuns(ctx context.Context, u effect.Units) error {
	state, err := u.State(ctx, "systemd-networkd.service")
	if err != nil {
		return err
	}
	if !state.Active {
		return errors.New("systemd-networkd is not running, so nothing would set up the cable: ws expects netplan to render with networkd, as on Ubuntu Server, not with NetworkManager, as on Ubuntu Desktop")
	}
	return nil
}
