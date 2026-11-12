package fans

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// refusedMark is in /run, so that a reboot clears it.
const refusedMark = "/run/ws-fans-refused"

// modules unloads Linux's own nct6683, which holds the chip as after sensors-detect, and loads
// nct6687. With Secure Boot on, the kernel refuses the driver until DKMS's key is enrolled, by
// hand at the console: docs/fans.md#secure-boot. Until then the run goes on, and the board's
// curves run. A refusal is marked in /run, which a reboot clears, as enrolling the key needs one,
// so that the runs between plan no load. Without systemd booted, as in CI's container, it loads nothing.
type modules struct {
	m   effect.Machine
	out effect.Out
}

// loaded reports whether the kernel holds the module.
func (c modules) loaded(name string) (bool, error) {
	_, err := c.m.Stat("/sys/module/" + name)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (c modules) Plan(context.Context) ([]host.Change, error) {
	if !host.Booted(c.m) {
		c.out.Say("fans' modules: skipped, systemd is not running")
		return nil, nil
	}
	var changes []host.Change
	if own, err := c.loaded("nct6683"); err != nil {
		return nil, err
	} else if own {
		changes = append(changes, command{c.m, "unload Linux's own driver of the chip, nct6683", []string{"modprobe", "-r", "nct6683"}})
	}
	if ours, err := c.loaded("nct6687"); err != nil {
		return nil, err
	} else if !ours {
		if _, err := c.m.Stat(refusedMark); errors.Is(err, fs.ErrNotExist) {
			changes = append(changes, load{c})
		} else if err != nil {
			return nil, err
		}
	}
	return changes, nil
}

// load loads nct6687, and notes the kernel's refusal without failing.
type load struct{ c modules }

func (load) String() string { return "load nct6687" }

func (l load) Apply(ctx context.Context) error {
	_, err := l.c.m.RunRoot(ctx, "modprobe", "nct6687")
	if err != nil && strings.Contains(err.Error(), "Key was rejected") {
		effect.SayLasting(l.c.out, "The kernel refused nct6687, as Secure Boot is on and DKMS's key is not enrolled, so the board's own curves drive its fans. docs/fans.md#secure-boot says how to enroll it.")
		_, err = l.c.m.RunRoot(ctx, "touch", "--", refusedMark)
		return err
	}
	return err
}
