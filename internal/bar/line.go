package bar

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"
)

// Bar is ws-bar: the host's load as a line.
type Bar struct {
	// FS is the machine's root, whose proc, sys and run the bar reads.
	FS fs.FS
	// Interval is the time between two lines, and between two of nvidia-smi's readings.
	Interval time.Duration

	cpu  *cpuTimes // the CPUs' reading of the line before
	gpus gpus
}

// Line returns the line of the host's load now. It reads the CPUs' use and temperature, the
// GPUs', the memory's, the NVMe drives' temperature, and "reboot" where one is due. A segment
// whose source gives nothing drops, and the CPUs' use shows from the second line on, being the
// use since the line before.
func (b *Bar) Line() string {
	var segments []string
	add := func(name string, use int, useOK bool, celsius int, celsiusOK bool) {
		s := name
		if useOK {
			s += fmt.Sprintf(" %d%%", use)
		}
		if celsiusOK {
			s += fmt.Sprintf(" %dC", celsius)
		}
		if useOK || celsiusOK {
			segments = append(segments, s)
		}
	}
	use, useOK := 0, false
	if now, err := readCPU(b.FS); err == nil {
		if b.cpu != nil {
			use, useOK = usage(*b.cpu, now)
		}
		b.cpu = &now
	}
	celsius, celsiusOK := hottest(b.FS, "coretemp", "Package id")
	if !celsiusOK {
		celsius, celsiusOK = hottest(b.FS, "k10temp", "Tctl")
	}
	add("cpu", use, useOK, celsius, celsiusOK)
	use, celsius, ok := b.gpus.reading(3 * b.Interval)
	add("gpu", use, ok, celsius, ok)
	use, err := readMemory(b.FS)
	celsius, celsiusOK = hottest(b.FS, "spd5118", "")
	add("ram", use, err == nil, celsius, celsiusOK)
	celsius, celsiusOK = hottest(b.FS, "nvme", "Composite")
	add("nvme", 0, false, celsius, celsiusOK)
	if _, err := fs.Stat(b.FS, path.Join("run", "reboot-required")); err == nil {
		segments = append(segments, "reboot")
	}
	return strings.Join(segments, "  ")
}
