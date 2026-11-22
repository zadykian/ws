package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// guardSpec is a unit of the kind guard: a check that plans nothing. It fails where a unit after it
// would lock the machine out, or would act on what a role refuses. It takes LoginKey with Home, or
// Command.
type guardSpec struct {
	base
	// LoginKey is a user who must hold a key that sshd takes, in Home/.ssh/authorized_keys, before
	// SSH by key alone.
	LoginKey string `toml:"login_key"`
	Home     string `toml:"home"`
	// Command is a role's own check, run as root on every machine, containers too: an argv whose
	// program is an absolute path. Its exit 0 passes, and any other fails the plan.
	Command []string `toml:"command"`
}

func (s guardSpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	var r resource.Resource
	switch {
	case len(s.Command) > 0 && s.LoginKey == "" && s.Home == "":
		if !path.IsAbs(s.Command[0]) {
			return nil, fmt.Errorf("command %q: its program is no absolute path", s.Command[0])
		}
		r = commandCheck{m: e.M, out: e.Out, argv: s.Command, said: new(bool)}
	case s.LoginKey != "" && len(s.Command) == 0:
		if err := checkUser("login_key", s.LoginKey); err != nil {
			return nil, err
		}
		if !path.IsAbs(s.Home) || path.Clean(s.Home) != s.Home || s.Home == "/" {
			return nil, fmt.Errorf("home %q is not a clean absolute path below /", s.Home)
		}
		r = keyCheck{m: e.M, sys: e.System, user: account.User{Name: s.LoginKey, Home: s.Home}}
	default:
		return nil, errors.New("a guard takes one of login_key with home and command")
	}
	return func(context.Context) ([]resource.Resource, error) { return []resource.Resource{r}, nil }, nil
}

// inContainer reports whether the machine is a container: Docker, Podman and systemd leave these
// marks. A container may boot systemd and run sshd, so the key guard does not skip one.
func inContainer(m effect.Machine) bool {
	for _, p := range []string{"/.dockerenv", "/run/.containerenv", "/run/systemd/container"} {
		if _, err := m.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// keyCheck fails where SSH by key alone would let the user in by no key. With passwords off, sshd
// takes only the keys in the user's authorized_keys.
type keyCheck struct {
	m    effect.Machine
	sys  System
	user account.User
}

// Plan fails where the home is not the user's in the system's accounts, on every machine. sshd reads
// the keys there, and the units of the user's files write there. It fails where authorized_keys, or
// a directory above it, is a link: the user owns the home and root reads the file. It also fails
// where the file is missing, holds no key, or is one sshd won't use, on every machine.
func (k keyCheck) Plan(ctx context.Context) ([]resource.Change, error) {
	home, err := k.sys.home(ctx, k.m, k.user.Name)
	if err != nil {
		return nil, err
	}
	if home != k.user.Home {
		return nil, fmt.Errorf("home %s is not %s's home, %s, as the system's accounts give it: "+
			"name that home, or none", k.user.Home, k.user.Name, home)
	}
	file := path.Join(k.user.Home, ".ssh/authorized_keys")
	keys, err := k.user.ReadHome(k.m, ".ssh/authorized_keys")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if !holdsKey(keys) {
		return nil, fmt.Errorf("%s holds no key, so SSH by key alone would let no one in: put the account's keys there first", file)
	}
	return nil, k.usable(file)
}

// holdsKey reports whether a line of the file is neither blank nor a comment.
func holdsKey(keys []byte) bool {
	for line := range strings.Lines(string(keys)) {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			return true
		}
	}
	return false
}

// usable fails where sshd would pass the file over, which root's cp leaves behind. sshd reads it as
// the user, and takes it only from that user or root, with no other able to write it. The home's
// owner is the user.
func (k keyCheck) usable(file string) error {
	home, err := k.m.RootAttrs(k.user.Home)
	if err != nil {
		return err
	}
	got, err := k.m.RootAttrs(file)
	if err != nil {
		return err
	}
	theirs := got.UID == home.UID || got.UID == 0 && got.Mode&0o004 != 0
	if theirs && got.Mode&0o022 == 0 {
		return nil
	}
	return fmt.Errorf("sshd would pass over %s, owner %d, mode %o, so SSH by key alone would let "+
		"no one in: give it to %s's owner, with mode 600", file, got.UID, got.Mode, k.user.Home)
}

// commandCheck is a role's own check: a command that fails where a later unit would act on what the
// role refuses.
type commandCheck struct {
	m    effect.Machine
	out  effect.Out
	argv []string
	// said keeps the lines to one telling a process, as a run plans a unit more than once.
	said *bool
}

// Plan runs the command as root. A pass says each line of its output. A failure fails the plan with
// the command's error, which holds its standard error, and its output.
func (c commandCheck) Plan(ctx context.Context) ([]resource.Change, error) {
	out, err := c.m.RunRoot(ctx, c.argv[0], c.argv[1:]...)
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text != "" {
			return nil, fmt.Errorf("%w: %s", err, text)
		}
		return nil, err
	}
	if !*c.said {
		for line := range strings.Lines(text) {
			if line = strings.TrimSpace(line); line != "" {
				c.out.Say("%s", line)
			}
		}
		*c.said = true
	}
	return nil, nil
}
