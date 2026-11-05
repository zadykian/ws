package docker

import (
	"context"
	"errors"
	"io/fs"
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// rootlessInstall runs Docker's tool as the user, which makes and starts the unit, enables it, and
// makes the rootless context. The tool refuses root, and refuses a user who can write the system's
// socket, which the docker group's removal, a step earlier, settles.
type rootlessInstall struct {
	m    effect.Machine
	user host.User
	uid  int
	out  effect.Out
}

func (r rootlessInstall) String() string { return "install " + r.user.Name + "'s rootless daemon" }

// Plan looks at the files alone, so a dry run on a new machine needs no user's systemd. A run cut
// between the tool's unit and its link runs the tool again, which keeps the unit.
func (r rootlessInstall) Plan(context.Context) ([]host.Change, error) {
	if !host.Booted(r.m) {
		r.out.Say("Docker rootless daemon: skipped, systemd is not running")
		return nil, nil
	}
	units := r.user.Home + "/.config/systemd/user"
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

// Apply runs the tool in the user's session.
func (r rootlessInstall) Apply(ctx context.Context) error {
	return r.user.Pass(ctx, r.m, "env", slices.Concat(host.SessionEnv(r.uid), []string{"dockerd-rootless-setuptool.sh", "install"})...)
}
