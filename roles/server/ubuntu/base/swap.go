package base

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// fstabLine swaps to the file at boot.
const fstabLine = "/swapfile none swap sw 0 0"

// swapSetting is the swap file's wanted size in MiB, nil for the RAM's, and the swappiness. The
// step checks the swappiness too, so that a wrong value stops the run with the swap unchanged.
type swapSetting struct {
	SizeMiB    *int
	Swappiness int
}

// swap reads the machine's swap and plans /swapfile from planSwap's result.
type swap struct {
	M      effect.Machine
	Out    effect.Out
	Disk   disk
	Want   swapSetting
	active bool // plans the swapon and the swapoff of the installer's file, not the file's own
}

// swapParts keeps /swapfile, base_swap_size_mb MiB, in /etc/fstab and swapped to; none where the
// size is 0. It takes the place of /swap.img, which Ubuntu Server's installer makes.
func swapParts(m effect.Machine, out effect.Out, d disk, want swapSetting) []host.Resource {
	s := swap{M: m, Out: out, Disk: d, Want: want}
	on := s
	on.active = true
	none := want.SizeMiB != nil && *want.SizeMiB == 0
	return []host.Resource{
		s,
		host.Line{M: m, Path: "/etc/fstab", Text: fstabLine, Match: `^/swapfile\s`, Absent: none, Mode: 0o644},
		on,
		host.Line{M: m, Path: "/etc/fstab", Text: installerImg + " none swap sw 0 0", Match: `^/swap\.img\s`, Absent: true},
		host.Absent{M: m, Path: installerImg},
	}
}

// state reads what the plan needs.
func (s swap) state() (swapState, error) {
	if sw := s.Want.Swappiness; sw < 0 || sw > 200 {
		return swapState{}, fmt.Errorf("base_swappiness is %d: it takes a whole number from 0 to 200", sw)
	}
	swaps, err := s.M.ReadFile("/proc/swaps")
	if err != nil {
		return swapState{}, err
	}
	areas, err := swapAreas(string(swaps))
	if err != nil {
		return swapState{}, err
	}
	info, err := s.M.ReadFile("/proc/meminfo")
	if err != nil {
		return swapState{}, err
	}
	avail, err := meminfo(string(info), "MemAvailable")
	if err != nil {
		return swapState{}, err
	}
	want, err := wantMiB(s.Want.SizeMiB, string(info))
	if err != nil {
		return swapState{}, err
	}
	free, err := s.Disk.Statfs("/")
	if err != nil {
		return swapState{}, err
	}
	file, err := s.Disk.Stat(swapFile)
	if err != nil {
		return swapState{}, err
	}
	return swapState{WantMiB: want, Areas: areas, AvailKiB: avail, DiskMiB: free, File: file, Booted: host.Booted(s.M)}, nil
}

// wantMiB returns the swap file's size: the setting, or the RAM rounded up to a whole GiB where it
// is unset.
func wantMiB(size *int, meminfoText string) (int64, error) {
	if size == nil {
		total, err := meminfo(meminfoText, "MemTotal")
		return ramSwapMiB(total), err
	}
	if *size < 0 {
		return 0, fmt.Errorf("base_swap_size_mb is %d: it takes a whole number of MiB, or 0 for none", *size)
	}
	return int64(*size), nil
}
