// Command ws sets up the Mac that works on the server.
package main

import (
	"os"

	"github.com/zadykian/ws/cmd"
)

func main() { os.Exit(cmd.Execute()) }
