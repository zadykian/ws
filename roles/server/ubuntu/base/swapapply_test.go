package base

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// applyOnce plans the group and applies its plan, and returns the commands run as root, but rm,
// and the disk's calls. The fake's /proc does not follow swapon, so a second plan is not asked.
func applyOnce(t *testing.T, m *effecttest.Machine, d *fakeDisk, want swapSetting) []string {
	g := swapGroup(m, d, want)
	p, err := g.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	var calls []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root ") && !strings.HasPrefix(c, "root rm ") {
			calls = append(calls, c)
		}
	}
	return append(calls, d.calls...)
}

func TestSwapOnANewMachine(t *testing.T) {
	m, d := booted(swapsImg), &fakeDisk{avail: 100000}
	calls := applyOnce(t, m, d, swapSetting{Swappiness: 10})
	// The RAM, 15932 MiB, rounded up to a whole GiB.
	want := []string{"root mkswap /swapfile.new", "root swapon /swapfile", "root swapoff /swap.img",
		"remove /swapfile.new", "allocate /swapfile.new 16384", "rename /swapfile.new /swapfile"}
	slices.Sort(calls)
	slices.Sort(want)
	if !slices.Equal(calls, want) {
		t.Errorf("calls = %q, want %q", calls, want)
	}
	if got := string(m.Files["/etc/fstab"]); got != "/dev/sda1 / ext4 defaults 0 1\n/swapfile none swap sw 0 0\n" {
		t.Errorf("fstab = %q", got)
	}
}

func TestSwapSizeZeroRemovesTheFile(t *testing.T) {
	m := booted("/swapfile file 4194300 0 -2\n")
	m.Files["/etc/fstab"] = []byte("/swapfile none swap sw 0 0\n")
	d := &fakeDisk{files: map[string]swapFileInfo{swapFile: {Exists: true, Size: 16384 * mib, Mode: 0o600}}}
	calls := applyOnce(t, m, d, swapSetting{SizeMiB: ptr(0), Swappiness: 10})
	if want := []string{"root swapoff /swapfile", "remove /swapfile"}; !slices.Equal(calls, want) {
		t.Errorf("calls = %q, want %q", calls, want)
	}
	if got := string(m.Files["/etc/fstab"]); got != "" {
		t.Errorf("fstab = %q, want no /swapfile line", got)
	}
}

func TestSwapWithoutSystemdSkipsSwapon(t *testing.T) {
	m, d := booted(""), &fakeDisk{avail: 100000}
	delete(m.Dirs, "/run/systemd/system")
	calls := applyOnce(t, m, d, swapSetting{SizeMiB: ptr(1024), Swappiness: 10})
	for _, c := range calls {
		if strings.HasPrefix(c, "root swap") {
			t.Errorf("ran %q without systemd", c)
		}
	}
	if !slices.Contains(calls, "rename /swapfile.new /swapfile") {
		t.Errorf("calls = %q, want the file made", calls)
	}
}
