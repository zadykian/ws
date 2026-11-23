package swap

import "fmt"

// planSwap checks that the memory and the disk have room, and returns what to change. The errors
// stop a run before it changes the swap. The disk's ends with hint.
func planSwap(s swapState, hint string) (swapSteps, error) {
	stale := s.stale()
	// swapoff brings what an area holds back into memory, and the OOM killer ends processes where
	// it does not fit. Half of the memory available leaves room for what runs meanwhile.
	off := []string{installerImg}
	if stale {
		off = []string{swapFile, installerImg}
	}
	for _, area := range off {
		if held, ok := s.Areas[area]; ok && held*2 > s.AvailKiB {
			return swapSteps{}, fmt.Errorf("%s holds %d MiB swapped out, and memory has %d MiB available. "+
				"Swapping it off needs twice what it holds: free memory, then run again", area, held/1024, s.AvailKiB/1024)
		}
	}
	create := s.WantMiB > 0 && (!s.File.Exists || stale)
	if create {
		// A file of another size is removed first, which frees its space.
		free := s.DiskMiB
		if stale {
			free += s.File.Size / mib
		}
		if free < s.WantMiB+diskSpareMiB {
			return swapSteps{}, fmt.Errorf("/ has %d MiB available, too little for a swap file of %d MiB and %d MiB more. %s",
				s.DiskMiB, s.WantMiB, diskSpareMiB, hint)
		}
	}
	_, onFile := s.Areas[swapFile]
	_, onImg := s.Areas[installerImg]
	return swapSteps{
		StopFile:      stale && onFile,
		RemoveFile:    stale,
		Create:        create,
		Restrict:      s.WantMiB > 0 && !create && s.File.Exists && (s.File.Mode.Perm() != 0o600 || s.File.UID != 0 || s.File.GID != 0),
		Swapon:        s.WantMiB > 0 && (create || !onFile),
		StopInstaller: onImg,
	}, nil
}
