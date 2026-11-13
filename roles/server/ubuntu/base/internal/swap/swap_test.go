package swap

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
)

// fakeDisk keeps the files a test gives it, and records the calls that change the disk.
type fakeDisk struct {
	files map[string]swapFileInfo
	avail int64
	calls []string
}

func (d *fakeDisk) Statfs(string) (int64, error)        { return d.avail, nil }
func (d *fakeDisk) Stat(p string) (swapFileInfo, error) { return d.files[p], nil }
func (d *fakeDisk) Restrict(p string) error             { return d.record("restrict " + p) }
func (d *fakeDisk) Rename(from, to string) error        { return d.record("rename " + from + " " + to) }
func (d *fakeDisk) Remove(p string) error               { return d.record("remove " + p) }
func (d *fakeDisk) Allocate(p string, size int64) error {
	return d.record(fmt.Sprintf("allocate %s %d", p, size))
}

func (d *fakeDisk) record(call string) error {
	d.calls = append(d.calls, call)
	return nil
}

const (
	meminfoText = "MemTotal:       16314752 kB\nMemAvailable:    8000000 kB\n"
	swapsHeader = "Filename Type Size Used Priority\n"
	swapsImg    = "/swap.img file 4194300 1024 -2\n"
)

// booted is a machine with the files that the swap step reads, and the installer's swap file.
func booted(swaps string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/proc/swaps"] = []byte(swapsHeader + swaps)
	m.Files["/proc/meminfo"] = []byte(meminfoText)
	m.Files["/etc/fstab"] = []byte("/dev/sda1 / ext4 defaults 0 1\n/swap.img none swap sw 0 0\n")
	for _, c := range []string{"mkswap /swapfile.new", "swapon /swapfile", "swapoff /swapfile", "swapoff /swap.img"} {
		m.Results[c] = effecttest.Result{}
	}
	return m
}

func swapGroup(m *effecttest.Machine, d *fakeDisk, want Setting) resource.Group {
	return resource.Group{Title: "Swap", Parts: swapParts(m, &effecttest.Out{}, d, want)}
}

func ptr(n int) *int { return &n }

func TestSwapSettled(t *testing.T) {
	m := booted("/swapfile file 4194300 0 -2\n")
	m.Files["/etc/fstab"] = []byte("/swapfile none swap sw 0 0\n")
	d := &fakeDisk{files: map[string]swapFileInfo{swapFile: {Exists: true, Size: 16384 * mib, Mode: 0o600}}, avail: 1}
	p, err := swapGroup(m, d, Setting{Swappiness: 10}).Plan(t.Context())
	if err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v, want none", p.Changes, err)
	}
}

func TestSwapRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		want  Setting
		avail int64
		text  string
	}{
		{"negative size", Setting{SizeMiB: ptr(-1)}, 100000, "base_swap_size_mb is -1"},
		{"swappiness too high", Setting{Swappiness: 201}, 100000, "base_swappiness is 201"},
		{"swappiness negative", Setting{Swappiness: -1}, 100000, "base_swappiness is -1"},
		{"disk too small", Setting{SizeMiB: ptr(1024)}, 3000, "too little for a swap file"},
	} {
		m, d := booted(""), &fakeDisk{avail: tt.avail}
		_, err := swapGroup(m, d, tt.want).Plan(t.Context())
		if err == nil || !strings.Contains(err.Error(), tt.text) {
			t.Errorf("%s: error = %v, want one with %q", tt.name, err, tt.text)
		}
		if len(d.calls) != 0 {
			t.Errorf("%s: changed the disk: %q", tt.name, d.calls)
		}
	}
}
