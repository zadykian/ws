package base

import "io/fs"

const (
	swapFile     = "/swapfile"
	installerImg = "/swap.img"
	mib          = 1 << 20
	// diskSpareMiB stays free besides the swap file, so that the machine's own writes still fit.
	diskSpareMiB = 2048
)

// swapFileInfo is what the swap file's path holds.
type swapFileInfo struct {
	Exists   bool
	Size     int64
	Mode     fs.FileMode
	UID, GID int
}

// swapState is everything that the swap's plan reads, so that the plan is a function of it.
type swapState struct {
	WantMiB  int64            // the swap file's size, and 0 for none
	Areas    map[string]int64 // the KiB each area swapped to holds swapped out, from /proc/swaps
	AvailKiB int64            // MemAvailable, from /proc/meminfo
	DiskMiB  int64            // the space available on /, from statfs
	File     swapFileInfo
	Booted   bool // swapon and swapoff need a booted systemd, as the role's tag said
}

// swapSteps are the changes that a state calls for, in the order they apply.
type swapSteps struct {
	StopFile      bool // swapoff /swapfile: the kernel refuses to remove a file it swaps to
	RemoveFile    bool
	Create        bool
	Restrict      bool // make the file root's, mode 0600: it holds what is swapped out of memory
	Swapon        bool
	StopInstaller bool // swapoff /swap.img, once /swapfile is on, so that what it holds can move there
}

// stale reports whether the swap file is of another size, which makes it again, not resizes it; at
// size 0 the step removes it.
func (s swapState) stale() bool { return s.File.Exists && s.File.Size != s.WantMiB*mib }
