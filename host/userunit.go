package host

import (
	"context"
	"errors"
	"io/fs"

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

// Plan leaves a unit with no file to whatever installs it. The next commit starts one with a file.
func (u UserUnit) Plan(ctx context.Context) ([]Change, error) {
	if !Booted(u.M) {
		u.Out.Say("%s's %s: skipped, systemd is not running", u.User.Name, u.Name)
		return nil, nil
	}
	_, err := u.M.Stat(u.User.Home + "/.config/systemd/user/" + u.Name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return nil, err
}
