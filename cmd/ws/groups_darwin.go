//go:build darwin

package main

// groups returns the command groups that exist on this system: ws mac, on a Mac.
func groups() []group {
	return []group{{name: "mac", summary: "the Mac's commands, run \"ws mac help\" for them", run: macCommand}}
}
