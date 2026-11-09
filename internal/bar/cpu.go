// Package bar is ws-bar, which ws server bar runs: a line of the host's load, which tmux's status
// line shows. See docs/tmux.md.
package bar

import (
	"bytes"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// cpuTimes is the time all CPUs spent since boot, busy and in all, in the kernel's ticks.
type cpuTimes struct{ busy, total uint64 }

// readCPU reads the first line of /proc/stat. Idle and iowait count as idle, as top counts them.
// The fields past steal, a guest's time, are counted in user and nice already.
func readCPU(fsys fs.FS) (cpuTimes, error) {
	data, err := fs.ReadFile(fsys, "proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	line, _, _ := bytes.Cut(data, []byte("\n"))
	fields := strings.Fields(string(line))
	if len(fields) < 6 || fields[0] != "cpu" {
		return cpuTimes{}, fmt.Errorf("/proc/stat starts with %q, not the CPUs' line", line)
	}
	var t cpuTimes
	for i, f := range fields[1:min(len(fields), 9)] {
		n, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuTimes{}, fmt.Errorf("/proc/stat: %w", err)
		}
		t.total += n
		if i != 3 && i != 4 {
			t.busy += n
		}
	}
	return t, nil
}

// usage returns the share of time busy from before to now, in percent. A counter that went back
// is no reading. iowait may go back, so the share is capped at 100.
func usage(before, now cpuTimes) (int, bool) {
	if now.total <= before.total || now.busy < before.busy {
		return 0, false
	}
	share := min((now.busy-before.busy)*100/(now.total-before.total), 100)
	return int(share), true
}
