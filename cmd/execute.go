// Package cmd holds ws's commands, which main.go runs by Execute.
package cmd

import (
	"context"
	"os"

	"github.com/zadykian/ws/engine/cli"
)

// Execute runs ws on the process's arguments and terminals, and returns its exit status.
func Execute() int {
	env := cli.NewEnv()
	return cli.Execute(context.Background(), cli.Root(env, groups(env)...), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}
