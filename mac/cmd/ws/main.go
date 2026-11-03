// Command ws sets up the Mac that works on the server.
package main

import (
	"fmt"
	"io"
	"os"
)

// group is a set of commands that exists on some systems alone, such as ws mac on a Mac.
type group struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

// usageOf returns the usage text, with a line for each group of this system.
func usageOf(groups []group) string {
	text := "usage: ws COMMAND\n\nCommands:\n  version  print the version of this ws\n"
	for _, g := range groups {
		text += "  " + g.name + "  " + g.summary + "\n"
	}
	return text
}

// run executes the command line args, writes to stdout and stderr, and returns the exit status.
func run(args []string, stdout, stderr io.Writer) int {
	usage := usageOf(groups())
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
	}
	for _, g := range groups() {
		if g.name == args[0] {
			return g.run(args[1:], stdout, stderr)
		}
	}
	_, _ = fmt.Fprintf(stderr, "ws: unknown command %q\n\n%s", args[0], usage)
	return 2
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
