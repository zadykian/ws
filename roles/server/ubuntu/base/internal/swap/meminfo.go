package swap

import (
	"fmt"
	"strconv"
	"strings"
)

// meminfo returns the KiB of one field of /proc/meminfo, such as MemTotal.
func meminfo(text, field string) (int64, error) {
	for _, line := range strings.Split(text, "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok || name != field {
			continue
		}
		kib, _, _ := strings.Cut(strings.TrimSpace(rest), " ")
		n, err := strconv.ParseInt(kib, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("/proc/meminfo %s: %w", field, err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("/proc/meminfo has no %s", field)
}

// ramSwapMiB is the swap file's default size in MiB: the RAM rounded up to a whole GiB. MemTotal
// in KiB is divided to MiB first, as Ansible's memtotal_mb was.
func ramSwapMiB(memTotalKiB int64) int64 { return (memTotalKiB/1024 + 1023) / 1024 * 1024 }
