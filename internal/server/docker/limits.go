package docker

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/server/values"
)

// WithLimits fills the limits of docker.slice that the values leave empty, with half the RAM and
// half the CPUs. That leaves the other half to the rest of the machine, the user's sessions among
// it.
// It reads the RAM from /proc/meminfo and the CPUs from /proc/cpuinfo.
func WithLimits(m effect.Machine, d values.Docker) (values.Docker, error) {
	if d.MemoryMax != "" && d.CPUQuota != "" {
		return d, nil
	}
	info, err := m.ReadFile("/proc/meminfo")
	if err != nil {
		return d, err
	}
	mb, err := memTotalMB(info)
	if err != nil {
		return d, err
	}
	cpus, err := m.ReadFile("/proc/cpuinfo")
	if err != nil {
		return d, err
	}
	n := 0
	for line := range bytes.Lines(cpus) {
		if bytes.HasPrefix(line, []byte("processor")) {
			n++
		}
	}
	if n == 0 {
		return d, fmt.Errorf("/proc/cpuinfo lists no processor")
	}
	if d.MemoryMax == "" {
		d.MemoryMax = strconv.Itoa(mb/2) + "M"
	}
	if d.CPUQuota == "" {
		d.CPUQuota = strconv.Itoa(n*50) + "%"
	}
	return d, nil
}

// memTotalMB returns MemTotal in whole MiB, as Ansible's memtotal_mb does.
func memTotalMB(meminfo []byte) (int, error) {
	for line := range strings.Lines(string(meminfo)) {
		if f := strings.Fields(line); len(f) == 3 && f[0] == "MemTotal:" && f[2] == "kB" {
			kb, err := strconv.Atoi(f[1])
			return kb / 1024, err
		}
	}
	return 0, fmt.Errorf("/proc/meminfo has no MemTotal in kB")
}
