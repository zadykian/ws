package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
)

// UserUnit keeps a unit of the user's own systemd running. effect.Units is the system's bus alone,
// so this runs systemctl --user, as the user. A unit with no file is left to whatever installs it.
type UserUnit struct {
	M    effect.Machine
	User User
	UID  int
	Out  effect.Out
	Name string
}

func (u UserUnit) systemctl(ctx context.Context, args ...string) ([]byte, error) {
	argv := slices.Concat(SessionEnv(u.UID), []string{"systemctl", "--user"}, args)
	return u.User.Output(ctx, u.M, "env", argv...)
}

// active asks the user's manager, where it is up, for the unit's state. The group plans before
// the manager's own start, so a manager that is down is no error. show exits with 0 in any state.
func (u UserUnit) active(ctx context.Context) (bool, error) {
	_, err := u.M.Stat(fmt.Sprintf("/run/user/%d/systemd/private", u.UID))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	out, err := u.systemctl(ctx, "show", "--property=ActiveState", "--value", u.Name)
	return strings.TrimSpace(string(out)) == "active", err
}

type userCall struct {
	u    UserUnit
	verb string
}

func (c userCall) String() string { return c.verb + " " + c.u.User.Name + "'s " + c.u.Name }

// Apply reloads the manager first, as the unit's file may be new.
func (c userCall) Apply(ctx context.Context) error {
	if _, err := c.u.systemctl(ctx, "daemon-reload"); err != nil {
		return err
	}
	_, err := c.u.systemctl(ctx, c.verb, c.u.Name)
	return err
}

// Plan starts the unit where it is down.
func (u UserUnit) Plan(ctx context.Context) ([]Change, error) {
	if !Booted(u.M) {
		u.Out.Say("%s's %s: skipped, systemd is not running", u.User.Name, u.Name)
		return nil, nil
	}
	_, err := u.M.Stat(u.User.Home + "/.config/systemd/user/" + u.Name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	active, err := u.active(ctx)
	if err != nil || active {
		return nil, err
	}
	return []Change{userCall{u, "start"}}, nil
}
