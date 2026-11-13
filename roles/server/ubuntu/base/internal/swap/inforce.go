package swap

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// inForce makes the swappiness in force the file's, with systemd-sysctl, which sets this one key as
// boot does. --strict fails where /proc/sys is read-only, instead of going on as if it were set.
type inForce struct {
	M          effect.Machine
	Out        effect.Out
	Swappiness int
}

type sysctl struct{ f inForce }

func (c sysctl) String() string { return fmt.Sprintf("set vm.swappiness to %d now", c.f.Swappiness) }

func (c sysctl) Apply(ctx context.Context) error {
	_, err := c.f.M.RunRoot(ctx, "/usr/lib/systemd/systemd-sysctl", "--strict", "--prefix=vm.swappiness")
	return err
}

// Plan reads /proc/sys/vm/swappiness, which a container can read though not write.
func (f inForce) Plan(context.Context) ([]resource.Change, error) {
	if !systemd.Booted(f.M) {
		f.Out.Say("swappiness: skipped, systemd is not running")
		return nil, nil
	}
	data, err := f.M.ReadFile("/proc/sys/vm/swappiness")
	if err != nil {
		return nil, err
	}
	now, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("/proc/sys/vm/swappiness: %w", err)
	}
	if now == f.Swappiness {
		return nil, nil
	}
	return []resource.Change{sysctl{f}}, nil
}
