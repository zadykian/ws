package docker

import (
	"context"
	"errors"
	"io/fs"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// rootlessInstall runs Docker's tool as the user, which makes and starts the unit, enables it, and
// makes the rootless context. The tool refuses root, and refuses a user who can write the system's
// socket, which the docker group's removal, a step earlier, settles.
type rootlessInstall struct {
	m   effect.Machine
	a   host.Account
	out effect.Out
}

func (r rootlessInstall) String() string { return "install " + r.a.Name + "'s rootless daemon" }

// Plan looks at the files alone, so a dry run on a new machine needs no user's systemd. A run cut
// between the tool's unit and its link runs the tool again, which keeps the unit.
func (r rootlessInstall) Plan(context.Context) ([]host.Change, error) {
	if !host.Booted(r.m) {
		r.out.Say("Docker rootless daemon: skipped, systemd is not running")
		return nil, nil
	}
	units := r.a.Home + "/.config/systemd/user"
	for _, p := range []string{units + "/docker.service", units + "/default.target.wants/docker.service"} {
		_, err := r.m.Stat(p)
		if errors.Is(err, fs.ErrNotExist) {
			return []host.Change{r}, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Apply reloads the user's systemd first: one that started before dbus-user-session landed does
// not know dbus.socket.
func (r rootlessInstall) Apply(ctx context.Context) error {
	env := host.SessionEnv(r.a.UID)
	if err := r.a.Pass(ctx, r.m, "env", slices.Concat(env, []string{"systemctl", "--user", "daemon-reload"})...); err != nil {
		return err
	}
	return r.a.Pass(ctx, r.m, "env", slices.Concat(env, []string{"dockerd-rootless-setuptool.sh", "install"})...)
}
