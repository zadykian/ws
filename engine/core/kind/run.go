package kind

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// systemPath is the PATH that a run state's commands get. The caller's, such as mise's shims, stays
// out, as a command may write it down, as Docker's setup tool does in the unit it makes.
const systemPath = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// runCommand is the run of a run state's command.
type runCommand struct{ r runAs }

func (c runCommand) String() string {
	return "run " + strings.Join(c.r.s.Command, " ") + " as " + c.r.s.User
}

func (c runCommand) Apply(ctx context.Context) error { return c.r.exec(ctx, true, c.r.s.Command) }

// exec runs a command line as the run's user, with the system's PATH. With pass set, it passes the
// output to ws's standard error, and otherwise drops it, as a check's exit status alone counts.
// Linux switches to the user by runuser, through the account, and gives it systemd's session,
// which systemctl --user needs. macOS has neither: there root's sudo switches, as ws runs as the
// user.
func (r runAs) exec(ctx context.Context, pass bool, line []string) error {
	if r.sys == MacOS {
		args := slices.Concat([]string{"-u", r.s.User, "-H", "env", systemPath}, line)
		if pass {
			return r.m.Pass(ctx, true, "sudo", args...)
		}
		_, err := r.m.RunRoot(ctx, "sudo", args...)
		return err
	}
	// root's home is /root, where ownerOf gives root none.
	a := account.Account{User: account.User{Name: "root", Home: "/root"}}
	if r.s.User != "root" {
		var err error
		if a, err = ownerOf(ctx, Env{M: r.m, System: r.sys}, r.s.User); err != nil {
			return err
		}
	}
	args := slices.Concat(systemd.SessionEnv(a.UID), []string{systemPath}, line)
	if pass {
		return a.Pass(ctx, r.m, "env", args...)
	}
	_, err := a.Output(ctx, r.m, "env", args...)
	return err
}

// Plan plans the command where a path of Creates is missing, or where Check fails. A check that
// cannot run fails too, and the command runs: the check's command is the run's to make work.
func (r runAs) Plan(ctx context.Context) ([]resource.Change, error) {
	if r.sys == Linux && !systemd.Booted(r.m) {
		r.out.Say("%s: skipped, systemd is not running", runCommand{r})
		return nil, nil
	}
	for _, p := range r.s.Creates {
		_, err := r.m.Stat(p)
		if errors.Is(err, fs.ErrNotExist) {
			return []resource.Change{runCommand{r}}, nil
		}
		if err != nil {
			return nil, err
		}
	}
	if len(r.s.Check) == 0 {
		return nil, nil
	}
	if err := r.exec(ctx, false, r.s.Check); err != nil {
		return []resource.Change{runCommand{r}}, nil //nolint:nilerr // a failed check is the run to do
	}
	return nil, nil
}
