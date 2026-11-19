package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// systemPath is the PATH that a run state's commands get, after its spec's path. The caller's, such
// as mise's shims, stays out, as a command may write it down in a file that outlives the run.
const systemPath = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// pathEnv is the run's PATH: its spec's directories, then the system's.
func (r runAs) pathEnv() string {
	if len(r.s.Path) == 0 {
		return systemPath
	}
	return "PATH=" + strings.Join(r.s.Path, ":") + ":" + strings.TrimPrefix(systemPath, "PATH=")
}

// runCommand is the run of a run state's command.
type runCommand struct{ r runAs }

func (c runCommand) String() string {
	return "run " + strings.Join(c.r.s.Command, " ") + " as " + c.r.s.User
}

func (c runCommand) Apply(ctx context.Context) error { return c.r.exec(ctx, true, c.r.s.Command) }

// exec runs a command line as the run's user, with its PATH (pathEnv). With pass set, it passes the
// output to ws's standard error, and otherwise drops it, as a check's exit status alone counts.
// Linux switches to the user by runuser, through the account, and gives it systemd's session,
// which systemctl --user needs, where systemd runs. macOS has neither (macExec).
func (r runAs) exec(ctx context.Context, pass bool, line []string) error {
	if r.sys == MacOS {
		return r.macExec(ctx, pass, line)
	}
	// root's home is /root, where ownerOf gives root none.
	a := account.Account{User: account.User{Name: "root", Home: "/root"}}
	if r.s.User != "root" {
		var err error
		if a, err = ownerOf(ctx, Env{M: r.m, System: r.sys}, r.s.User); err != nil {
			return err
		}
	}
	// runuser passes the caller's env on. A run with no session drops the caller's session from it:
	// root's runtime directory is closed to the user, and dotnet fails on it.
	session := []string{"-u", "XDG_RUNTIME_DIR", "-u", "DBUS_SESSION_BUS_ADDRESS"}
	if systemd.Booted(r.m) && (!r.s.Unbooted || r.hasRuntimeDir(a.UID)) {
		session = systemd.SessionEnv(a.UID)
	}
	args := slices.Concat(session, []string{r.pathEnv()}, line)
	if pass {
		return a.Pass(ctx, r.m, "env", args...)
	}
	_, err := a.Output(ctx, r.m, "env", args...)
	return err
}

// hasRuntimeDir says whether the user's session has made its runtime directory. An unbooted run
// needs no session, and gets its env only where that directory exists. dotnet fails on a missing
// XDG_RUNTIME_DIR, as before the account's first login or lingering.
func (r runAs) hasRuntimeDir(uid int) bool {
	_, err := r.m.Stat(fmt.Sprintf("/run/user/%d", uid))
	return err == nil
}

// macExec runs a command line on the Mac, where ws runs as the user and sudo alone switches. A
// check, run with pass unset, runs as ws's own user, as a read must ask no password. A check of
// root's paths reads what the Mac keeps readable by all. A command runs as ws's user with no sudo,
// as root through sudo, or as another user through root's sudo.
func (r runAs) macExec(ctx context.Context, pass bool, line []string) error {
	argv := slices.Concat([]string{"env", r.pathEnv()}, line)
	root := false
	switch {
	case !pass || r.s.User == r.self:
	case r.s.User == "root":
		root = true
	default:
		argv, root = slices.Concat([]string{"sudo", "-u", r.s.User, "-H"}, argv), true
	}
	if pass {
		return r.m.Pass(ctx, root, argv[0], argv[1:]...)
	}
	_, err := r.m.Run(ctx, argv[0], argv[1:]...)
	return err
}

// Plan plans the command where a path of Creates is missing, or where Check fails. A check that
// cannot run fails too, and the command runs: the check's command is the run's to make work.
func (r runAs) Plan(ctx context.Context) ([]resource.Change, error) {
	if r.sys == Linux && !r.s.Unbooted && !systemd.Booted(r.m) {
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
