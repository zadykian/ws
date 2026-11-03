//go:build darwin

package main

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
)

// groups returns the command groups that exist on this system: ws mac, on a Mac.
func groups(p process) []*cobra.Command {
	p.machine = effect.OSMachine{}
	return []*cobra.Command{newMacCmd(p)}
}
