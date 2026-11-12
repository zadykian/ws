package wizard

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
)

// file is mac.conf.
func (c *Cable) file() confFile { return confFile{c.Conf, cableStage, c.M, c.Out} }

// Apply takes ws's port of the cable, where none holds ws's end, from WS_CABLE_PORT or the
// terminal, the one with carrier by default. link names it in /etc/ws/host_vars.yml on ws. Then ws
// server apply --only network,wireguard runs there, as both roles read it: the network role must
// give it 10.77.0.1 as ws's end. After that, service takes the Mac's service. none, as either
// answer, writes an empty WS_CABLE_SERVICE. Where mac.conf has a service, again takes the stage's
// steps.
func (c *Cable) Apply(ctx context.Context, _ step.Plan) error {
	cf, err := c.Ws.want()
	if err != nil {
		return err
	}
	if !c.lacks() {
		return c.again(ctx, cf.CablePort)
	}
	if v, set := c.Env("WS_CABLE_SERVICE"); set && v == "" {
		return c.write(ctx, "")
	}
	if err := c.read(ctx, ""); err != nil {
		return err
	}
	if c.end() == "" {
		var out []string
		for _, p := range c.ports {
			if p.carrier == "1" {
				out = append(out, p.name+" (carrier)")
			} else {
				out = append(out, p.name)
			}
		}
		port, err := c.ask(ctx, "WS_CABLE_PORT", "ws's port of the cable, of "+strings.Join(out, ", "),
			one(c.up()), c.names())
		if err != nil {
			return err
		}
		if port == "" {
			return c.write(ctx, "")
		}
		c.Out.Say("%s: %s goes in /etc/ws/host_vars.yml on ws, and ws server apply --only network,wireguard "+
			"runs there in a unit of its own", cableStage, port)
		if err := c.name(ctx, port); err != nil {
			return err
		}
		if err := c.read(ctx, port); err != nil {
			return err
		}
		if c.end() != port {
			return fmt.Errorf("%s: ws's %s lacks %s after ws server apply --only network,wireguard: plug the "+
				"Mac's cable into it, or mend what that printed, then run the wizard again", cableStage, port, cableEnd)
		}
	}
	return c.service(ctx)
}

// name puts the port in ws's /etc/ws/host_vars.yml and runs the network and wireguard roles there.
func (c *Cable) name(ctx context.Context, port string) error {
	return c.Ws.nameCable(ctx, port)
}

// holds tells whether ws's status names the port as the cable's.
func (c *Cable) holds(ctx context.Context, port string) bool {
	s, ok := c.Ws.status(ctx)
	return ok && s.Cable == port
}

// again names WS_CABLE_PORT in /etc/ws/host_vars.yml on ws where ws's status names none or another.
// That holds after ws's install, a failed roles' run or a change in mac.conf. Then it runs the
// roles there, with no question and no dock. It stops on a port that ws lacks.
func (c *Cable) again(ctx context.Context, port string) error {
	s, ok := c.Ws.status(ctx)
	if !ok {
		return fmt.Errorf("%s: status failed on ws", cableStage)
	}
	if err := c.read(ctx, ""); err != nil {
		return err
	}
	if !slices.Contains(c.names(), port) {
		return fmt.Errorf("%s: WS_CABLE_PORT=%s names none of ws's wired ports, %s: set it to one", cableStage,
			port, strings.Join(c.names(), ", "))
	}
	if s.Cable == port {
		return nil
	}
	c.Out.Say("%s: %s goes in /etc/ws/host_vars.yml on ws again, and ws server apply --only network,wireguard "+
		"runs there in a unit of its own", cableStage, port)
	return c.name(ctx, port)
}

// service takes the Mac's service of the cable, from WS_CABLE_SERVICE in the environment or the
// terminal. The default is the one service with a 169.254 address, which an adapter gives itself
// where no DHCP server answers. It writes it in mac.conf, and runs ws mac apply, which gives it
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
	if _, err := c.MacApply(ctx, false); err != nil {
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
