package cli

import (
	"context"
	"fmt"
	"path"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/macos/facts"
	"github.com/zadykian/ws/macos/internal/keys"
	"github.com/zadykian/ws/macos/roles/cable"
	"github.com/zadykian/ws/macos/roles/cld"
	"github.com/zadykian/ws/macos/roles/networks"
	"github.com/zadykian/ws/macos/values"
)

// The Mac's parts that are no file: each a command of ws that a run state's unit runs, as the
// user, with --check as its check. Each reads as the user, and a write of root's goes through sudo.

// newCldCmd returns ws macos cld, which cld's unit runs.
func newCldCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Procedure("cld", "install cld into ~/.local/bin, and keep it at its latest release",
		"It installs cld with the install.sh of its latest release where it is missing, and runs cld "+
			"update where a newer release is out. A cld built from a clone is left as it is. A failed "+
			"install or update, or a latest release that it cannot read, is a warning.",
		func(_ context.Context, out effect.Out) ([]resource.Resource, error) {
			home, err := homeOf(p)
			if err != nil {
				return nil, err
			}
			return []resource.Resource{&cld.Cld{Home: home, M: p.Machine, Out: out}}, nil
		})
}

// newKeysCmd returns ws macos keys.
func newKeysCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Procedure("keys", "copy the .pub of each key of Secretive's that values.yml names into /etc/ws",
		"ssh and git name a key that Secretive's agent holds by its .pub, which macOS keeps ws from "+
			"reading in Secretive's container. So ws keeps a copy, from the agent, in /etc/ws/secretive. "+
			"A key in a file has none.",
		func(ctx context.Context, out effect.Out) ([]resource.Resource, error) {
			home, err := homeOf(p)
			if err != nil {
				return nil, err
			}
			c, err := values.LoadMac(p.Machine, values.Path(p.Getenv, p.Pwd))
			if err != nil {
				return nil, err
			}
			return keys.Copies(ctx, p.Machine, out, c, home)
		})
}

// newCableCmd returns ws macos cable, which the cable's unit runs.
func newCableCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Procedure("cable", "give the cable's service its Manual Configuration of the Mac's end",
		"The service that a server's WS_CABLE_SERVICE names takes 10.77.0.2/30 with no router, through "+
			"sudo, where networksetup shows other settings. A service networksetup lacks, or one with a "+
			"router outside the cable's /30, as an uplink has, fails it.",
		func(_ context.Context, out effect.Out) ([]resource.Resource, error) {
			servers, err := serversOf(p)
			var rs []resource.Resource
			for _, s := range servers {
				if s.CableService != "" {
					rs = append(rs, cable.Cable{Conf: s.Conf, M: p.Machine, Out: out}.Part())
				}
			}
			return rs, err
		})
}

// newNetworksCmd returns ws macos networks, which the networks' unit runs.
func newNetworksCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Procedure("networks", "route each server's WS_TUNNEL_NETWORKS and send its WS_RESOLVERS' domains to it",
		"Each server with WS_PUBLIC_KEY gets its routes.d/values, and a file in /etc/resolver for each "+
			"domain of its WS_RESOLVERS, written through sudo. A file of ws's in /etc/resolver that no "+
			"server names goes. A file of a domain that ws did not write fails it.",
		func(_ context.Context, out effect.Out) ([]resource.Resource, error) {
			servers, err := serversOf(p)
			if err != nil {
				return nil, err
			}
			return []resource.Resource{networks.Mac{Servers: servers, M: p.Machine, Out: out}}, nil
		})
}

// serversOf reads each server's values, checked as the units' facts are.
func serversOf(p enginecli.Env) ([]facts.Server, error) {
	return facts.Servers(p.Machine, values.Path(p.Getenv, p.Pwd))
}

// homeOf returns the user's home from HOME.
func homeOf(p enginecli.Env) (string, error) {
	home := p.Getenv("HOME")
	if !path.IsAbs(home) {
		return "", fmt.Errorf("HOME %q names no home: run it from the user's own shell", home)
	}
	return home, nil
}
