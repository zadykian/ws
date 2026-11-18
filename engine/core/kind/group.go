package kind

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// gid returns the group that the file's group or group_of names, and whether it names one.
func (f filePart) gid(ctx context.Context, m effect.Machine) (int, bool, error) {
	switch {
	case f.Group != "":
		id, err := groupID(m, f.Group)
		return id, true, err
	case f.GroupOf != "":
		a, err := account.User{Name: f.GroupOf}.Lookup(ctx, m)
		return a.GID, true, err
	}
	return 0, false, nil
}

// groupID returns the id of the group name from /etc/group. A file's unit reads it as it plans, as
// a package that mise installs first may make the group.
func groupID(m effect.Machine, name string) (int, error) {
	text, err := m.ReadFile("/etc/group")
	if err != nil {
		return 0, err
	}
	for line := range strings.Lines(string(text)) {
		f := strings.Split(strings.TrimSpace(line), ":")
		if len(f) >= 3 && f[0] == name {
			id, err := strconv.Atoi(f[2])
			if err != nil {
				return 0, fmt.Errorf("the id of the group %s is %q: %w", name, f[2], err)
			}
			return id, nil
		}
	}
	return 0, fmt.Errorf("the group %s is missing from /etc/group", name)
}
