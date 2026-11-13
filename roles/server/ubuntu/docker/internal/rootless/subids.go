package rootless

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// minSubIDs is the range a container needs for the users inside it.
const minSubIDs = 65536

// subIDs fails where the user has no range of subordinate ids, which rootlesskit maps a container's
// users to. useradd gives one to each new user, so only a hand-made account misses it.
type subIDs struct {
	m    effect.Machine
	user string
}

// Plan reads both files, and fails on the first with no range.
func (s subIDs) Plan(context.Context) ([]resource.Change, error) {
	for _, file := range []string{"/etc/subuid", "/etc/subgid"} {
		data, err := s.m.ReadFile(file)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if !hasRange(string(data), s.user) {
			return nil, fmt.Errorf("%s gives %s no range of %d: run usermod --add-subuids 100000-165535 --add-subgids 100000-165535 %s first",
				file, s.user, minSubIDs, s.user)
		}
	}
	return nil, nil
}

func hasRange(data, user string) bool {
	for line := range strings.Lines(data) {
		f := strings.Split(strings.TrimSpace(line), ":")
		if len(f) != 3 || f[0] != user {
			continue
		}
		if n, err := strconv.Atoi(f[2]); err == nil && n >= minSubIDs {
			return true
		}
	}
	return false
}
