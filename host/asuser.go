package host

import (
	"context"

	"github.com/zadykian/ws/effect"
)

// User is a user of the machine and the home that commands run as it get.
type User struct {
	Name string
	Home string
}

// argv returns the command that runs name as the user with the user's HOME, and whether it needs
// root. The caller's HOME would otherwise reach the command, as the user's login does not run.
// root itself needs no switch of user, and a container has no sudo.
func (u User) argv(name string, args []string) (cmd string, argv []string, root bool) {
	env := append([]string{"HOME=" + u.Home, name}, args...)
	if u.Name == "root" {
		return "env", env, false
	}
	return "runuser", append([]string{"-u", u.Name, "--", "env"}, env...), true
}

// Pass runs the command as the user and passes its output to ws's standard error.
func (u User) Pass(ctx context.Context, m effect.Machine, name string, args ...string) error {
	cmd, argv, root := u.argv(name, args)
	return m.Pass(ctx, root, cmd, argv...)
}

// Output runs the command as the user and returns its standard output.
func (u User) Output(ctx context.Context, m effect.Machine, name string, args ...string) ([]byte, error) {
	cmd, argv, root := u.argv(name, args)
	if root {
		return m.RunRoot(ctx, cmd, argv...)
	}
	return m.Run(ctx, cmd, argv...)
}
