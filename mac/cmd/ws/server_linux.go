//go:build linux

package main

import (
	"context"
	"io"
	"os"
	"runtime/debug"

	"github.com/zadykian/ws"
	"github.com/zadykian/ws/mac/internal/effect"
)

// groups returns the command groups that exist on this system: ws server, on Linux.
func groups() []group {
	return []group{{name: "server", summary: "the server's commands, run \"ws server help\" for them", run: runServer}}
}

func runServer(args []string, stdout, stderr io.Writer) int {
	info, ok := debug.ReadBuildInfo()
	return serverCommand(context.Background(), args, os.Geteuid(), ws.Playbook, versionOf(info, ok), effect.OSMachine{},
		stdout, stderr)
}
