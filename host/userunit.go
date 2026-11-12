package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// UserUnit keeps a unit of the user's own systemd running, and restarts it where any of Watch has a
// change to make, as Unit does for the system's. effect.Units is the system's bus alone, so this
// runs systemctl --user, as the user. A unit with no file is left to whatever installs it.
type UserUnit struct {
	M       effect.Machine
	Account Account
	Out     effect.Out
	Name    string
	Watch   []Resource
}

func (u UserUnit) mark() mark { return mark{u.M, u.Account.Name + "-" + u.Name} }

func (u UserUnit) systemctl(ctx context.Context, args ...string) ([]byte, error) {
	argv := slices.Concat(SessionEnv(u.Account.UID), []string{"systemctl", "--user"}, args)
	return u.Account.Output(ctx, u.M, "env", argv...)
}

// active asks the user's manager, where the manager is up, for the unit's state. The group plans
// before the manager's own start, so a manager that is down is no error. show exits with 0 in any
// state.
func (u UserUnit) active(ctx context.Context) (bool, error) {
	_, err := u.M.Stat(fmt.Sprintf("/run/user/%d/systemd/private", u.Account.UID))
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
	// forced marks a restart, which Group.Apply still applies after a failure.
	forced bool
}

func (c userCall) Forced() bool { return c.forced }

func (c userCall) String() string { return c.verb + " " + c.u.Account.Name + "'s " + c.u.Name }

func (userCall) Kind() step.ChangeKind { return step.Restart }

// Apply reloads the manager first, as a watched file may be one of the unit's.
func (c userCall) Apply(ctx context.Context) error {
	if _, err := c.u.systemctl(ctx, "daemon-reload"); err != nil {
		return err
	}
	if _, err := c.u.systemctl(ctx, c.verb, c.u.Name); err != nil {
		return err
	}
	return c.u.mark().settle(ctx)
}

// Plan starts a unit that is down, which serves any watched change too, and otherwise
// restarts it for a watched change or a mark that an earlier run left.
func (u UserUnit) Plan(ctx context.Context) ([]Change, error) {
	if !Booted(u.M) {
		u.Out.Say("%s's %s: skipped, systemd is not running", u.Account.Name, u.Name)
		return nil, nil
	}
	_, err := u.M.Stat(u.Account.Home + "/.config/systemd/user/" + u.Name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	active, err := u.active(ctx)
	if err != nil {
		return nil, err
	}
	if !active {
		return []Change{userCall{u: u, verb: "start"}}, nil
	}
	k := u.mark()
	owed, err := k.owed()
	if err != nil {
		return nil, err
	}
	changed, err := anyChange(ctx, u.Watch)
	if err != nil {
		return nil, err
	}
	var cs []Change
	if changed && !owed {
		cs = append(cs, owe{k})
	}
	if changed || owed {
		cs = append(cs, userCall{u: u, verb: "restart", forced: true})
	}
	return cs, nil
}

// Report reads the unit's state, stopped where the manager is down. Plan says that systemd is not
// running, so this doesn't.
func (u UserUnit) Report(ctx context.Context) ([]step.Item, error) {
	item := step.Item{Kind: step.ServiceItem, Name: u.Account.Name + "'s " + u.Name, Want: "running", Found: noSystemd}
	if !Booted(u.M) {
		return []step.Item{item}, nil
	}
	active, err := u.active(ctx)
	if err != nil {
		return nil, err
	}
	item.Found = "stopped"
	if active {
		item.Found = "running"
	}
	return []step.Item{item}, nil
}
