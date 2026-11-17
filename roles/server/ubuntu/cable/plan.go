package cable

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/server/ubuntu/cable/internal/netplan"
)

// Plan checks the port and netplan's files, then plans the file, and the reload that a change to it
// calls for. The values' check has refused a port that no interface's name takes, and a bad
// address.
func (d direct) Plan(ctx context.Context) ([]resource.Change, error) {
	if d.Interface == "" {
		return d.reloaded(ctx, file.Absent{M: d.M, Path: netplan.Path})
	}
	if _, err := d.M.Stat("/sys/class/net/" + d.Interface); errors.Is(err, fs.ErrNotExist) {
		d.Out.Say("this machine has no interface %s, so the cable stays as it is; WS_CABLE_PORT names the port of the Mac's cable, as ip link lists it", d.Interface)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	rendered, err := netplan.Render(d.Interface, d.Address)
	if err != nil {
		return nil, err
	}
	port, err := readPort(ctx, d.M, d.Interface, d.Address)
	if err != nil {
		return nil, err
	}
	problems, err := netplan.Check(ctx, d.M, rendered, port)
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, " "))
	}
	if err := networkdRuns(ctx, d.M, d.U); err != nil {
		return nil, err
	}
	return d.reloaded(ctx, file.File{M: d.M, Path: netplan.Path, Content: rendered, Mode: 0o600})
}
