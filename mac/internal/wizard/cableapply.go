package wizard

import (
	"context"
	"fmt"
	"strings"
)

// file is mac.conf.
func (c *Cable) file() confFile { return confFile{c.Host.Conf, cableStage, c.M, c.Out} }

// service takes the Mac's service of the cable, from WS_CABLE_SERVICE in the environment or the
// terminal. The default is the one service with a 169.254 address, which an adapter gives itself
// where no DHCP server answers. It writes it in mac.conf, and runs ws setup, which gives it
// 10.77.0.2. Then ping must reach ws's end over it. none writes an empty WS_CABLE_SERVICE.
func (c *Cable) service(ctx context.Context) error {
	out, err := c.M.Run(ctx, "networksetup", "-listallnetworkservices")
	if err != nil {
		return fmt.Errorf("%s: networksetup -listallnetworkservices failed", cableStage)
	}
	var list, linked []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimPrefix(l, "*"); l != "" && !strings.HasPrefix(l, "An asterisk (*) denotes") {
			list = append(list, l)
		}
	}
	for _, s := range list {
		if info, err := c.M.Run(ctx, "networksetup", "-getinfo", s); err == nil &&
			strings.Contains("\n"+string(info), "\nIP address: 169.254.") {
			linked = append(linked, s)
		}
	}
	value, err := c.ask(ctx, "WS_CABLE_SERVICE", "The Mac's service of the cable, of "+strings.Join(list, ", "),
		one(linked), list)
	if err != nil {
		return err
	}
	if err := c.write(ctx, value); err != nil || value == "" {
		return err
	}
	if _, err := c.Setup.Run(ctx, false); err != nil {
		return err
	}
	if _, err := c.M.Run(ctx, "ping", "-c", "1", "-t", "3", cableEnd); err != nil {
		return fmt.Errorf("%s: ping -c 1 -t 3 %s, ws's end of the cable, got no answer: mac.conf's "+
			"WS_CABLE_SERVICE must name the service of the adapter that the cable plugs into", cableStage, cableEnd)
	}
	return nil
}

// write writes value as mac.conf's WS_CABLE_SERVICE, in double quotes, after ws's end as
// WS_CABLE_PORT, or empty for none, which the wizard never asks again.
func (c *Cable) write(ctx context.Context, value string) error {
	lines := "WS_CABLE_SERVICE=\n"
	if value != "" {
		lines = "WS_CABLE_PORT=" + c.end() + "\nWS_CABLE_SERVICE=\"" + value + "\"\n"
	}
	return c.file().add(ctx, lines)
}
