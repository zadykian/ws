package network

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/host"
)

// Plan checks the address, the port and netplan's files, then plans the file, and the reload that
// a change to it calls for.
func (d direct) Plan(ctx context.Context) ([]host.Change, error) {
	if d.Interface == "" {
		return d.reloaded(ctx, host.Absent{M: d.M, Path: netplanPath})
	}
	if strings.ContainsAny(d.Interface, "/ \t\n") {
		return nil, fmt.Errorf("network_direct_link_interface %q is not an interface's name", d.Interface)
	}
	if _, err := d.M.Stat("/sys/class/net/" + d.Interface); errors.Is(err, fs.ErrNotExist) {
		d.Out.Say("this machine has no interface %s, so the direct link stays as it is; network_direct_link_interface names the port of the Mac's cable, as ip link lists it", d.Interface)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	if problems := AddressProblems(d.Address); len(problems) > 0 {
		return nil, fmt.Errorf("network_direct_link_address: %s", strings.Join(problems, " "))
	}
	file, err := Render(d.Interface, d.Address)
	if err != nil {
		return nil, err
	}
	port, err := d.checkPort(ctx)
	if err != nil {
		return nil, err
	}
	problems, err := checkNetplan(ctx, d.M, file, port)
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, " "))
	}
	if host.Booted(d.M) {
		if state, err := d.U.State(ctx, "systemd-networkd.service"); err != nil {
			return nil, err
		} else if !state.Active {
			return nil, errors.New("systemd-networkd is not running, so nothing would set up the direct link: ws expects netplan to render with networkd, as on Ubuntu Server, not with NetworkManager, as on Ubuntu Desktop")
		}
	}
	return d.reloaded(ctx, host.File{M: d.M, Path: netplanPath, Content: file, Mode: 0o600})
}
