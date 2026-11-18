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
// would lock the machine out, or would act on what a role refuses. It takes LoginKey with Home,
// SudoRight, or Command. A container, which has no login to lose, passes the first two.
type guardSpec struct {
	base
	// LoginKey is a user who must hold a key that sshd takes, in Home/.ssh/authorized_keys, before
	// SSH by key alone.
	LoginKey string `toml:"login_key"`
	Home     string `toml:"home"`
	// SudoRight is a user who must have a way to root by sudo, before root's login closes: the
	// right, and a password for sudo to ask. root needs none.
	SudoRight string `toml:"sudo_right"`
	// Command is a role's own check, run as root on every machine, containers too: an argv whose
	// program is an absolute path. Its exit 0 passes, and any other fails the plan.
	Command []string `toml:"command"`
}

func (s guardSpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	var r resource.Resource
	switch {
	case len(s.Command) > 0 && s.LoginKey == "" && s.SudoRight == "" && s.Home == "":
		if !path.IsAbs(s.Command[0]) {
			return nil, fmt.Errorf("command %q: its program is no absolute path", s.Command[0])
		}
		r = commandCheck{m: e.M, out: e.Out, argv: s.Command, said: new(bool)}
	case s.LoginKey != "" && s.SudoRight == "" && len(s.Command) == 0:
		if err := checkUser("login_key", s.LoginKey); err != nil {
			return nil, err
		}
		if !path.IsAbs(s.Home) || path.Clean(s.Home) != s.Home || s.Home == "/" {
			return nil, fmt.Errorf("home %q is not a clean absolute path below /", s.Home)
		}
		r = keyCheck{m: e.M, user: account.User{Name: s.LoginKey, Home: s.Home}}
	case s.SudoRight != "" && s.LoginKey == "" && s.Home == "" && len(s.Command) == 0:
		if err := checkUser("sudo_right", s.SudoRight); err != nil {
			return nil, err
		}
		r = sudoRight{m: e.M, user: s.SudoRight}
	default:
		return nil, errors.New("a guard takes one of login_key with home, sudo_right and command")
	}
	return func(context.Context) ([]resource.Resource, error) { return []resource.Resource{r}, nil }, nil
}

// inContainer reports whether the machine is a container, which has no SSH to lose: Docker, Podman
// and systemd leave these marks.
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
	user account.User
}

// Plan fails where authorized_keys, or a directory above it, is a link: the user owns the home and
// root reads the file. It also fails where the file is missing, holds no key, or is one sshd won't
// use, outside a container.
func (k keyCheck) Plan(context.Context) ([]resource.Change, error) {
	if inContainer(k.m) {
		return nil, nil
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

// sudoRight fails where closing root's login would leave the user no way to root: the user can't
// run sudo, or has no password for it to ask.
type sudoRight struct {
	m    effect.Machine
	user string
}

// Plan asks sudo for the user's rights, and passwd for the password's state.
func (s sudoRight) Plan(ctx context.Context) ([]resource.Change, error) {
	if s.user == "root" || inContainer(s.m) {
		return nil, nil
	}
	list, err := s.m.RunRoot(ctx, "sudo", "-l", "-U", s.user)
	if err != nil {
		return nil, err
	}
	switch {
	case !strings.Contains(string(list), "may run the following commands"):
		return nil, fmt.Errorf("%s has no right to sudo, so closing root's login would leave no way to root: "+
			"add it to the sudo group first", s.user)
	case strings.Contains(string(list), "NOPASSWD"):
		return nil, nil
	}
	status, err := s.m.RunRoot(ctx, "passwd", "--status", s.user)
	if err != nil {
		return nil, err
	}
	if f := strings.Fields(string(status)); len(f) < 2 || f[1] != "P" {
		return nil, fmt.Errorf("%s has no password for sudo to ask, so closing root's login would leave no way "+
			"to root: run passwd %s first", s.user, s.user)
	}
	return nil, nil
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
