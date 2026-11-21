package kind

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// gid returns the group that the file's group names, and whether it names one.
func (f filePart) gid(m effect.Machine) (int, bool, error) {
	if f.Group == "" {
		return 0, false, nil
	}
	id, err := groupID(m, f.Group)
	return id, true, err
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
