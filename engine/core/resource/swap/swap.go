// Package swap keeps a swap file, /swapfile, in place of the one that Ubuntu Server's installer
// makes, /swap.img.
package swap

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

// fstabLine swaps to the file at boot.
const fstabLine = "/swapfile none swap sw 0 0"

// File is /swapfile, of SizeMiB MiB, in /etc/fstab and swapped to, and none where the size is 0. A
// nil size is the RAM rounded up to a whole GiB. It takes the place of /swap.img. Hint ends the
// refusal of a size that the disk lacks room for: how to give another.
type File struct {
	M       effect.Machine
	SizeMiB *int
	Hint    string
}

// Plan plans the file, its line in /etc/fstab, the swap to it, and the installer's file's removal,
// on the disk of the machine that ws runs on.
func (f File) Plan(ctx context.Context) ([]resource.Change, error) {
	return f.parts(osDisk{}).Plan(ctx)
}

// Report reports the parts.
func (f File) Report(ctx context.Context) ([]step.Item, error) { return f.parts(osDisk{}).Report(ctx) }

// swap reads the machine's swap and plans /swapfile from planSwap's result. It holds the File's
// fields, not the File, whose Report would make it a part of itself.
type swap struct {
	M       effect.Machine
	SizeMiB *int
	Hint    string
	Disk    disk
	active  bool // plans the swapon and the swapoff of the installer's file, not the file's own
}

// parts keeps the file on the disk d.
func (f File) parts(d disk) resource.Joined {
	s := swap{M: f.M, SizeMiB: f.SizeMiB, Hint: f.Hint, Disk: d}
	on := s
	on.active = true
	none := f.SizeMiB != nil && *f.SizeMiB == 0
	return resource.Joined{
		s,
		file.Line{M: f.M, Path: "/etc/fstab", Text: fstabLine, Match: `^/swapfile\s`, Absent: none, Mode: 0o644},
		on,
		file.Line{M: f.M, Path: "/etc/fstab", Text: installerImg + " none swap sw 0 0", Match: `^/swap\.img\s`, Absent: true},
		file.Absent{M: f.M, Path: installerImg},
	}
}

// state reads what the plan needs.
func (s swap) state() (swapState, error) {
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
	want, err := wantMiB(s.SizeMiB, string(info))
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
	return swapState{WantMiB: want, Areas: areas, AvailKiB: avail, DiskMiB: free, File: file}, nil
}

// wantMiB returns the swap file's size: the setting, or the RAM rounded up to a whole GiB where it
// is unset.
func wantMiB(size *int, meminfoText string) (int64, error) {
	if size == nil {
		total, err := meminfo(meminfoText, "MemTotal")
		return ramSwapMiB(total), err
	}
	if *size < 0 {
		return 0, fmt.Errorf("the swap file's size is %d MiB: it takes a whole number of MiB, or 0 for none", *size)
	}
	return int64(*size), nil
}
