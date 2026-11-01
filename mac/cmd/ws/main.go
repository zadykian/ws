// Command ws sets up the Mac that works on the server.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/zadykian/ws/mac/internal/effect"
)

const usage = "usage: ws COMMAND\n\nCommands:\n  version  print the version of this ws\n  conf [--lines] [FILE]  print mac.conf's checked settings as shell assignments\n  setup [--check] STEP  set up one part of the Mac, as ws setup does\n  tunnel run     keep the tunnel to ws up, as root\n  tunnel status  print the way in use and the peer, as root\n  tunnel key     make the Mac's key where none is, and print the public key, as root\n  tunnel loaded  succeed where launchd has the daemon loaded\n  tunnel routes [--check] NAME [CIDR...]  route the networks into the tunnel, or drop NAME\n  wizard [--check] [--update]  set the Mac up in stages\n  resolver [--check] DOMAIN [ADDRESS...]  send DOMAIN's names to the nameservers, or drop it\n"

// run executes the command line args, writes to stdout and stderr, and returns the exit status.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stdout, usage)
		return 0
	}
	switch args[0] {
	case "-h", "--help", "help":
		_, _ = io.WriteString(stdout, usage)
		return 0
	case "version":
		return versionCommand(stdout)
	case "conf":
		return confCommand(args[1:], effect.OSMachine{}, os.Getenv, stdout, stderr)
	case "setup":
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
	_, _ = fmt.Fprintf(stderr, "ws: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
