package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

const noCachePath = "/etc/sudoers.d/ws-no-cache"

// sudoParts makes the step's parts: the check of ws_user's way to root, where another user holds
// it, then the drop-in that ends sudo's cache. visudo reads it before it
// is in place, though a broken one would not lock anyone out: sudo-rs skips the file with an error.
// sudo skips a name with a dot, so the file under check has no effect.
func sudoParts(m effect.Machine, user string) ([]resource.Resource, error) {
	conf, err := files.ReadFile("files/ws-no-cache")
	if err != nil {
		return nil, err
	}
	dropIn := file.Validated{
		File: file.File{M: m, Path: noCachePath, Content: conf, Mode: 0o440},
		Check: func(path string) (string, []string) {
			return "visudo", []string{"-cf", path}
		},
	}
	if user == "root" {
		return []resource.Resource{dropIn}, nil
	}
	return []resource.Resource{sudoRight{m, user}, dropIn}, nil
}

// sudoRight fails where closing root's login would leave no way to root: the user can't run sudo,
// or has no password for it to ask. It plans nothing, so its failure ends the run before sshd's
// step. A container has no login to lose, and CI's user no right.
type sudoRight struct {
	m    effect.Machine
	user string
}

func (s sudoRight) Plan(ctx context.Context) ([]resource.Change, error) {
	if inContainer(s.m) {
		return nil, nil
	}
	list, err := s.m.RunRoot(ctx, "sudo", "-l", "-U", s.user)
	if err != nil {
		return nil, err
	}
	switch {
	case !strings.Contains(string(list), "may run the following commands"):
		return nil, fmt.Errorf("%s has no right to sudo, so closing root's login would leave no way to root: add it to the sudo group first", s.user)
	case strings.Contains(string(list), "NOPASSWD"):
		return nil, nil
	}
	status, err := s.m.RunRoot(ctx, "passwd", "--status", s.user)
	if err != nil {
		return nil, err
	}
	if f := strings.Fields(string(status)); len(f) < 2 || f[1] != "P" {
		return nil, fmt.Errorf("%s has no password for sudo to ask, so closing root's login would leave no way to root: run passwd %s first", s.user, s.user)
	}
	return nil, nil
}
