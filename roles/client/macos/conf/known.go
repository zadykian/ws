package conf

import (
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
)

// Known returns the names of the servers that a run lists, sorted with no repeats. They are those
// of servers/ beside mac.conf at mac, and current if servers/ lacks it. With no servers/, the list
// holds current alone.
func Known(m effect.Machine, mac, current string) ([]string, error) {
	names, err := Servers(m, mac)
	if err != nil {
		return nil, err
	}
	if current != "" && !slices.Contains(names, current) {
		names = append(names, current)
	}
	slices.Sort(names)
	return names, nil
}
