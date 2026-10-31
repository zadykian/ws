// Command ws sets up the Mac that works on the server.
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = "usage: ws COMMAND\n\nCommands:\n  tunnel run  keep the tunnel to ws up, as root\n"

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
	case "tunnel":
		return tunnel(args[1:], stdout, stderr)
	}
	_, _ = fmt.Fprintf(stderr, "ws: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
