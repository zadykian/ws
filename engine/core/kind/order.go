package kind

import (
	"fmt"
	"slices"
	"strings"
)

// ordered returns the units' names in the order they run: by name, each unit moved behind the
// one it waits on, ahead of the next by name. A unit that waits on one no spec holds, or a loop,
// is an error.
func ordered(all map[string]unitKind) ([]string, error) {
	names := make([]string, 0, len(all))
	for n := range all {
		names = append(names, n)
	}
	slices.Sort(names)
	waiting := map[string][]string{}
	var roots []string
	for _, n := range names {
		after := all[n].waits()
		if after == "" {
			roots = append(roots, n)
			continue
		}
		if _, ok := all[after]; !ok {
			return nil, fmt.Errorf("unit %s waits on %s, which no spec holds", n, after)
		}
		waiting[after] = append(waiting[after], n)
	}
	order := make([]string, 0, len(names))
	var place func(n string)
	place = func(n string) {
		order = append(order, n)
		for _, w := range waiting[n] {
			place(w)
		}
	}
	for _, n := range roots {
		place(n)
	}
	if len(order) < len(names) {
		var left []string
		for _, n := range names {
			if !slices.Contains(order, n) {
				left = append(left, n+" (after "+all[n].waits()+")")
			}
		}
		return nil, fmt.Errorf("units wait in a loop: %s", strings.Join(left, ", "))
	}
	return order, nil
}
