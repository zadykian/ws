package docker_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/server/docker"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func procMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/proc/meminfo"] = []byte("MemTotal:       16384512 kB\nMemFree: 5 kB\n")
	m.Files["/proc/cpuinfo"] = []byte("processor\t: 0\nmodel name\t: x\n\nprocessor\t: 1\nmodel name\t: x\n\n" +
		"processor\t: 2\n\nprocessor\t: 3\n")
	return m
}

func TestWithLimitsIsHalfTheRAMAndTheCPUs(t *testing.T) {
	got, err := docker.WithLimits(procMachine(), values.Default().Docker)
	if err != nil {
		t.Fatal(err)
	}
	if got.MemoryMax != "8000M" || got.CPUQuota != "200%" {
		t.Errorf("limits = %q, %q; want 8000M and 200%%", got.MemoryMax, got.CPUQuota)
	}
}

func TestWithLimitsKeepsTheValuesThatAreSet(t *testing.T) {
	d := values.Default().Docker
	d.MemoryMax, d.CPUQuota = "1G", "50%"
	got, err := docker.WithLimits(effecttest.NewMachine(), d)
	if err != nil || got.MemoryMax != "1G" || got.CPUQuota != "50%" {
		t.Errorf("WithLimits = %+v, %v; want the values as they were, with no read", got, err)
	}
}

func TestWithLimitsFailsOnAnUnreadableMachine(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/proc/meminfo"] = []byte("MemFree: 5 kB\n")
	if _, err := docker.WithLimits(m, values.Default().Docker); err == nil {
		t.Error("a /proc/meminfo with no MemTotal passed")
	}
}
