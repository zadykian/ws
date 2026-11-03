package main

import (
	"os"
	"runtime/debug"

	"github.com/zadykian/ws/effect"
)

// process is what the commands take from the process they run in. main builds it from os, and the
// tests build it from fakes.
type process struct {
	machine effect.Machine
	getenv  func(string) string
	euid    int
	version string
}

// newProcess returns the process of this ws.
func newProcess() process {
	info, ok := debug.ReadBuildInfo()
	return process{machine: effect.OSMachine{}, getenv: os.Getenv, euid: os.Geteuid(), version: versionOf(info, ok)}
}
