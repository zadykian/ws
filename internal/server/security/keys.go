package security

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

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

// Plan fails where authorized_keys is missing, holds no key, or is a file sshd won't use, outside a
// container.
func (k keyCheck) Plan(context.Context) ([]host.Change, error) {
	if inContainer(k.m) {
		return nil, nil
	}
	file := path.Join(k.home, ".ssh/authorized_keys")
	keys, err := k.m.ReadRootFile(file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if !holdsKey(keys) {
		return nil, fmt.Errorf("%s holds no key, so SSH by key only would let no one in. "+
			"Put the account's keys there first, as the README's \"A new machine\" says", file)
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

// usable fails where sshd would pass the file over, which root's cp leaves behind: sshd reads it as
// the user, and takes it only from that user or root, with no other able to write it. The home's
// owner is the user.
func (k keyCheck) usable(file string) error {
	home, err := k.m.RootAttrs(k.home)
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
	return fmt.Errorf("sshd would pass over %s, owner %d, mode %o, so SSH by key only would let "+
		"no one in: give it to %s's owner, with mode 600", file, got.UID, got.Mode, k.home)
}
