package bar

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// readMemory returns the share of memory in use, in percent. MemAvailable, which the kernel can
// hand out without swapping, is the rest.
func readMemory(fsys fs.FS) (int, error) {
	data, err := fs.ReadFile(fsys, "proc/meminfo")
	if err != nil {
		return 0, err
	}
	kib := map[string]uint64{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		if n, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			kib[strings.TrimSuffix(fields[0], ":")] = n
		}
	}
	total, available := kib["MemTotal"], kib["MemAvailable"]
	if total == 0 || available > total {
		return 0, fmt.Errorf("/proc/meminfo holds no MemTotal and MemAvailable")
	}
	return int((total - available) * 100 / total), nil //nolint:gosec // at most 100, as available <= total
}
