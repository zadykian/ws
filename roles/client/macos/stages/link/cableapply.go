package link

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// file is the server's file.
func (c *Cable) file() values.File {
	return values.File{Path: c.Conf, Scope: values.ServerFile, Stage: stage.Cable, M: c.M, Out: c.Out}
}

// Apply takes ws's port of the cable, where none holds ws's end, from WS_CABLE_PORT or the
// terminal, the one with carrier by default. link names it in /etc/ws/values.yml on ws. Then ws
// server ubuntu apply --only cable,tunnel runs there, as both roles read it: the cable role
// must give it 10.77.0.1 as ws's end. After that, service takes the Mac's service. none, as either
// answer, writes an empty WS_CABLE_SERVICE. Where the server's file has a service, again takes the stage's
// steps.
func (c *Cable) Apply(ctx context.Context, _ step.Plan) error {
	cf, err := values.Read(c.M, c.Conf)
	if err != nil {
		return err
	}
	if !c.lacks() {
		return c.again(ctx, cf.CablePort)
	}
	if c.owner() != "" {
		return c.write(ctx, "")
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
		port, err := c.ask(ctx, "WS_CABLE_PORT", c.SSH.Host+"'s port of the cable", one(c.up()), c.names(), out)
		if err != nil {
			return err
		}
		if port == "" {
			return c.write(ctx, "")
		}
		c.Out.Say("%s: %s goes in /etc/ws/values.yml on %s, and ws server ubuntu apply --only cable,tunnel "+
			"runs there in a unit of its own", stage.Cable, port, c.SSH.Host)
		if err := c.name(ctx, port); err != nil {
			return err
		}
		if err := c.read(ctx, port); err != nil {
			return err
		}
		if c.end() != port {
			return fmt.Errorf("%s: %s's %s lacks %s after ws server ubuntu apply --only cable,tunnel: plug the "+
				"Mac's cable into it, or mend what that printed, then run the wizard again", stage.Cable, c.SSH.Host, port, cableEnd)
		}
	}
	return c.service(ctx)
}

// name puts the port in ws's /etc/ws/values.yml and runs the cable and tunnel roles there.
func (c *Cable) name(ctx context.Context, port string) error {
	return c.Ws.NameCable(ctx, port)
}

// holds tells whether ws's status names the port as the cable's.
func (c *Cable) holds(ctx context.Context, port string) bool {
	s, ok := c.Ws.Status(ctx)
	return ok && s.Cable == port
}

// again names WS_CABLE_PORT in /etc/ws/values.yml on ws where ws's status names none or another.
// That holds after ws's install, a failed roles' run or a change in the server's file. Then it runs the
// roles there, with no question and no dock. It stops on a port that ws lacks.
func (c *Cable) again(ctx context.Context, port string) error {
	s, ok := c.Ws.Status(ctx)
	if !ok {
		return fmt.Errorf("%s: status failed on %s", stage.Cable, c.SSH.Host)
	}
	if err := c.read(ctx, ""); err != nil {
		return err
	}
	if !slices.Contains(c.names(), port) {
		return fmt.Errorf("%s: WS_CABLE_PORT=%s names none of %s's wired ports, %s: set it to one", stage.Cable,
			port, c.SSH.Host, strings.Join(c.names(), ", "))
	}
	if s.Cable == port {
		return nil
	}
	c.Out.Say("%s: %s goes in /etc/ws/values.yml on %s again, and ws server ubuntu apply --only cable,tunnel "+
		"runs there in a unit of its own", stage.Cable, port, c.SSH.Host)
	return c.name(ctx, port)
}

// service takes the Mac's service of the cable, from WS_CABLE_SERVICE in the environment or the
// terminal. The default is the one service with a 169.254 address, which an adapter gives itself
// where no DHCP server answers. It writes it in the server's file, and runs ws client macos apply, which
// gives it 10.77.0.2. Then ping must reach ws's end over it. none writes an empty WS_CABLE_SERVICE.
func (c *Cable) service(ctx context.Context) error {
	out, err := c.M.Run(ctx, "networksetup", "-listallnetworkservices")
	if err != nil {
		return fmt.Errorf("%s: networksetup -listallnetworkservices failed", stage.Cable)
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
	value, err := c.ask(ctx, "WS_CABLE_SERVICE", "The Mac's service of the cable", one(linked), list, list)
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
		return fmt.Errorf("%s: ping -c 1 -t 3 %s, %s's end of the cable, got no answer: server.conf's "+
			"WS_CABLE_SERVICE must name the service of the adapter that the cable plugs into", stage.Cable, cableEnd, c.SSH.Host)
	}
	return nil
}

// write adds value as the server file's WS_CABLE_SERVICE, or empty for none, which the wizard never
// asks again, after ws's end as WS_CABLE_PORT. A port that the file has stays.
func (c *Cable) write(ctx context.Context, value string) error {
	lines := values.Text("WS_CABLE_SERVICE", "")
	if value != "" {
		lines = values.Text("WS_CABLE_PORT", c.end(), "WS_CABLE_SERVICE", value)
	}
	return c.file().Add(ctx, lines)
}
