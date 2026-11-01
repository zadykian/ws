//go:build linux

package main

import (
	"context"
	"io"
	"os"
	"runtime/debug"

	"github.com/zadykian/ws"
	"github.com/zadykian/ws/effect"
)

// groups returns the command groups that exist on this system: ws server, on Linux.
func groups() []group {
	return []group{{name: "server", summary: "the server's commands, run \"ws server help\" for them", run: runServer}}
}

// runServer runs as root, which serverCommand checks, so the machine's root commands go without
// sudo: a container has none.
func runServer(args []string, stdout, stderr io.Writer) int {
	info, ok := debug.ReadBuildInfo()
	return serverCommand(context.Background(), args, os.Geteuid(), ws.Playbook, versionOf(info, ok),
		effect.OSMachine{Sudo: []string{}}, os.Stdin, stdout, stderr)
}
