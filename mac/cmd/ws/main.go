// Command ws sets up the Mac that works on the server.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/zadykian/ws/mac/internal/effect"
)

const usage = "usage: ws COMMAND\n\nCommands:\n  conf [--lines] [FILE]  print mac.conf's checked settings as shell assignments\n  setup [--check] STEP  set up one part of the Mac, as setup.sh does\n  tunnel run     keep the tunnel to ws up, as root\n  tunnel status  print the way in use and the peer, as root\n  tunnel key     make the Mac's key where none is, and print the public key, as root\n"

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
	case "conf":
		return confCommand(args[1:], effect.OSMachine{}, os.Getenv, stdout, stderr)
	case "setup":
		return setupCommand(context.Background(), args[1:], os.Geteuid(), effect.OSMachine{}, os.Getenv, stderr)
	case "tunnel":
		return tunnel(args[1:], stdout, stderr)
	}
	_, _ = fmt.Fprintf(stderr, "ws: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
