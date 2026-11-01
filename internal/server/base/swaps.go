package base

import (
	"fmt"
	"strconv"
	"strings"
)

// swapAreas parses /proc/swaps, whose size and used columns are in KiB, and returns the KiB each
// area holds swapped out, by its path. The kernel escapes a space in a path as \040.
func swapAreas(swaps string) (map[string]int64, error) {
	areas := map[string]int64{}
	for i, line := range strings.Split(swaps, "\n") {
		f := strings.Fields(line)
		if i == 0 && len(f) > 0 && f[0] == "Filename" || len(f) == 0 {
			continue
		}
		if len(f) < 5 {
			return nil, fmt.Errorf("/proc/swaps line %d has %d columns, want 5", i+1, len(f))
		}
		used, err := strconv.ParseInt(f[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("/proc/swaps line %d: %w", i+1, err)
		}
		areas[strings.ReplaceAll(f[0], `\040`, " ")] = used
	}
	return areas, nil
}
