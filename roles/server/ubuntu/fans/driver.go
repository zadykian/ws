package fans

import (
	"context"
	"errors"
	"io/fs"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// dkmsBinary is where the dkms package puts the command.
const dkmsBinary = "/usr/sbin/dkms"

// driver builds the driver for the running kernel by DKMS, which builds it again for each new
// kernel, from the headers linux-headers-generic brings with it. Another version in DKMS would
// stand in the pinned one's way, so it goes with its source.
type driver struct{ m effect.Machine }

// command runs a program as root.
type command struct {
	m    effect.Machine
	text string
	args []string
}

func (c command) String() string { return c.text }

func (c command) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, c.args[0], c.args[1:]...)
	return err
}

// Plan reads DKMS's versions of the driver. Before dkms is installed, as in a dry run, it holds
// none.
func (d driver) Plan(ctx context.Context) ([]host.Change, error) {
	kernel, err := kernelRelease(ctx, d.m)
	if err != nil {
		return nil, err
	}
	var state dkmsState
	if _, err := d.m.Stat(dkmsBinary); err == nil {
		out, err := d.m.RunRoot(ctx, "dkms", "status", "nct6687d")
		if err != nil {
			return nil, err
		}
		state = parseDKMS(string(out), nct6687Short, kernel)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var changes []host.Change
	for _, v := range state.versions {
		if v == nct6687Short {
			continue
		}
		changes = append(changes,
			command{d.m, "remove the driver's version " + v + " from DKMS", []string{"dkms", "remove", "nct6687d/" + v, "--all"}},
			command{d.m, "remove the source of version " + v, []string{"rm", "-rf", "--", "/usr/src/nct6687d-" + v}})
	}
	if !slices.Contains(state.versions, nct6687Short) {
		changes = append(changes, command{d.m, "add the driver to DKMS", []string{"dkms", "add", "nct6687d/" + nct6687Short}})
	}
	if !state.installed {
		changes = append(changes, command{d.m, "build and install the driver for " + kernel,
			[]string{"dkms", "install", "nct6687d/" + nct6687Short, "-k", kernel}})
	}
	return changes, nil
}
