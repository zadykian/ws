package wireguard

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// networkGroup is the group networkd reads its files as.
const networkGroup = "systemd-network"

// groupID returns the id of systemd-network from /etc/group. The group comes with systemd.
func groupID(m effect.Machine) (int, error) {
	text, err := m.ReadFile("/etc/group")
	if err != nil {
		return 0, err
	}
	for line := range strings.Lines(string(text)) {
		f := strings.Split(strings.TrimSpace(line), ":")
		if len(f) >= 3 && f[0] == networkGroup {
			id, err := strconv.Atoi(f[2])
			if err != nil {
				return 0, fmt.Errorf("the id of the group %s is %q: %w", networkGroup, f[2], err)
			}
			return id, nil
		}
	}
	return 0, fmt.Errorf("the group %s is missing: systemd creates it", networkGroup)
}
