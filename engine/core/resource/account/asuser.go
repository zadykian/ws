package account

import (
	"context"
	"fmt"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
)

// User is a user of the machine. Its commands run as root, from root's home, whatever Home says.
type User struct {
	Name string
	Home string
}

// SystemPath is the PATH that a command of ws's own gets: the system's. The caller's, such as
// mise's shims, stays out, as the command may write it down in a file that outlives the run.
const SystemPath = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// argv returns the command that runs name as root with root's HOME. The caller's HOME would
// otherwise reach the command, as no login runs. env -C starts it in root's home.
//
// The command starts from a cleared environment, with HOME and SystemPath alone. ws's own may hold
// a secret input's value, which a command such as cld, claude or git must not see: the secrets'
// task keeps it. Another user than root fails, as ws runs commands as root alone.
func (u User) argv(name string, args []string) ([]string, error) {
	if u.Name != "root" {
		return nil, fmt.Errorf("run %s as %s: ws runs commands as root alone", name, u.Name)
	}
	return slices.Concat([]string{"-i", "-C", "/root", "HOME=/root", SystemPath, name}, args), nil
}

// Pass runs the command as root and passes its output to ws's standard error.
func (u User) Pass(ctx context.Context, m effect.Machine, name string, args ...string) error {
	argv, err := u.argv(name, args)
	if err != nil {
		return err
	}
	return m.Pass(ctx, false, "env", argv...)
}

// Output runs the command as root and returns its standard output.
func (u User) Output(ctx context.Context, m effect.Machine, name string, args ...string) ([]byte, error) {
	argv, err := u.argv(name, args)
	if err != nil {
		return nil, err
	}
	return m.Run(ctx, "env", argv...)
}

// PassInput runs the command as root with stdin as its standard input, and passes its output to
// ws's standard error.
func (u User) PassInput(ctx context.Context, m effect.Machine, stdin []byte, name string, args ...string) error {
	argv, err := u.argv(name, args)
	if err != nil {
		return err
	}
	return m.PassInput(ctx, stdin, "env", argv...)
}
