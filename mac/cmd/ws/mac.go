package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/zadykian/ws/effect"
)

const macUsage = "usage: ws mac COMMAND\n\nCommands:\n  conf [--lines] [FILE]  print mac.conf's checked settings as shell assignments\n  apply [--check] STEP  set up one part of the Mac\n  tunnel run     keep the tunnel to ws up, as root\n  tunnel status  print the way in use and the peer, as root\n  tunnel key     make the Mac's key where none is, and print the public key, as root\n  tunnel loaded  succeed where launchd has the daemon loaded\n  tunnel routes [--check] NAME [CIDR...]  route the networks into the tunnel, or drop NAME\n  wizard [--check] [--update]  set the Mac up in stages\n  resolver [--check] DOMAIN [ADDRESS...]  send DOMAIN's names to the nameservers, or drop it\n"

// macCommand runs the commands of ws mac, the Mac's side of ws. Only the entry of ws mac in the
// table of groups is built for darwin alone: this function is built everywhere, so that the tests
// run on Linux.
func macCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stdout, macUsage)
		return 0
	}
	switch args[0] {
	case "-h", "--help", "help":
		_, _ = io.WriteString(stdout, macUsage)
		return 0
	case "conf":
		return confCommand(args[1:], effect.OSMachine{}, os.Getenv, stdout, stderr)
	case "apply":
		return setupCommand(context.Background(), args[1:], os.Geteuid(), effect.OSMachine{}, os.Getenv, stdout, stderr)
	case "tunnel":
		if len(args) > 1 && args[1] == "loaded" {
			return tunnelLoaded(context.Background(), args[2:], effect.OSMachine{}, stderr)
		}
		if len(args) > 1 && args[1] == "routes" {
			return tunnelRoutes(context.Background(), args[2:], effect.OSMachine{}, stderr)
		}
		return tunnel(args[1:], stdout, stderr)
	case "wizard":
		return wizardMain(args[1:], stderr)
	case "resolver":
		return resolverCommand(context.Background(), args[1:], effect.OSMachine{}, stderr)
	}
	_, _ = fmt.Fprintf(stderr, "ws mac: unknown command %q\n\n%s", args[0], macUsage)
	return 2
}
