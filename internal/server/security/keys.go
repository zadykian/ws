package security

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

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

// keyCheck refuses to turn passwords off with no key to log in with. With passwords off, SSH lets
// in only the keys in the user's authorized_keys, and Ubuntu's installer puts its key with the
// installer's user, not root's.
type keyCheck struct {
	m    effect.Machine
	home string
}

// Plan fails where authorized_keys is missing or empty, outside a container.
func (k keyCheck) Plan(context.Context) ([]host.Change, error) {
	if inContainer(k.m) {
		return nil, nil
	}
	file := path.Join(k.home, ".ssh/authorized_keys")
	keys, err := k.m.ReadRootFile(file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%s is missing or empty, so SSH by key only would let no one in. "+
			"Copy the installer's user's keys there first, as the README's \"A new machine\" says", file)
	}
	return nil, nil
}
