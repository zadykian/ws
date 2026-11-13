package swap

import (
	"strings"
	"testing"
)

func TestPlanSwap(t *testing.T) {
	for _, tt := range []struct {
		name string
		s    swapState
		want swapSteps
	}{
		{"new machine with the installer's file",
			swapState{WantMiB: 8192, Areas: map[string]int64{installerImg: 100}, AvailKiB: 1 << 22, DiskMiB: 50000, Booted: true},
			swapSteps{Create: true, Swapon: true, StopInstaller: true}},
		{"settled", swapState{WantMiB: 8192, Areas: map[string]int64{swapFile: 5}, File: file8, DiskMiB: 1, Booted: true}, swapSteps{}},
		{"file made, not swapped to", swapState{WantMiB: 8192, File: file8, Booted: true}, swapSteps{Swapon: true}},
		{"a container makes the file and skips swapon and swapoff",
			swapState{WantMiB: 8192, Areas: map[string]int64{installerImg: 0}, DiskMiB: 50000}, swapSteps{Create: true}},
		{"another size is made again",
			swapState{WantMiB: 4096, Areas: map[string]int64{swapFile: 10}, AvailKiB: 1 << 22, DiskMiB: 100, File: file8, Booted: true},
			swapSteps{StopFile: true, RemoveFile: true, Create: true, Swapon: true}},
		{"size 0 removes the file", swapState{File: file8, Areas: map[string]int64{swapFile: 0}, Booted: true},
			swapSteps{StopFile: true, RemoveFile: true}},
		{"size 0 with nothing", swapState{Booted: true}, swapSteps{}},
		{"mode too open", swapState{WantMiB: 8192, File: swapFileInfo{Exists: true, Size: 8192 * mib, Mode: 0o644},
			Areas: map[string]int64{swapFile: 0}}, swapSteps{Restrict: true}},
		{"owner not root", swapState{WantMiB: 8192, File: swapFileInfo{Exists: true, Size: 8192 * mib, Mode: 0o600, UID: 1000},
			Areas: map[string]int64{swapFile: 0}}, swapSteps{Restrict: true}},
		{"disk with exactly the room", swapState{WantMiB: 100, DiskMiB: 100 + diskSpareMiB}, swapSteps{Create: true}},
		{"the stale file's space counts", swapState{WantMiB: 10000, DiskMiB: 4100, File: file8},
			swapSteps{RemoveFile: true, Create: true}},
		{"memory with exactly the room", swapState{WantMiB: 0, Areas: map[string]int64{installerImg: 512}, AvailKiB: 1024}, swapSteps{}},
	} {
		got, err := planSwap(tt.s)
		if err != nil || got != tt.want {
			t.Errorf("%s: plan = %+v, %v, want %+v", tt.name, got, err, tt.want)
		}
	}
}

func TestPlanSwapRefusals(t *testing.T) {
	for _, tt := range []struct {
		name, want string
		s          swapState
	}{
		{"installer's file holds too much", "/swap.img holds 1 MiB swapped out, and memory has 1 MiB available",
			swapState{Areas: map[string]int64{installerImg: 1024}, AvailKiB: 1024}},
		{"stale file holds too much", "/swapfile holds 4 MiB",
			swapState{WantMiB: 1, File: file8, Areas: map[string]int64{swapFile: 4096}, AvailKiB: 4096}},
		{"disk lacks room", "/ has 2147 MiB available, too little for a swap file of 100 MiB and 2048 MiB more",
			swapState{WantMiB: 100, DiskMiB: 2147}},
		{"disk lacks room for a bigger file", "too little",
			swapState{WantMiB: 20000, DiskMiB: 4100, File: file8}},
	} {
		if _, err := planSwap(tt.s); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error = %v, want one with %q", tt.name, err, tt.want)
		}
	}
}

func TestPlanSwapSettledNeedsNoRoom(t *testing.T) {
	// A file already of the wanted size needs no room, and swapping to it holds nothing to move.
	s := swapState{WantMiB: 8192, File: file8, Areas: map[string]int64{swapFile: 1 << 30}, Booted: true}
	if got, err := planSwap(s); err != nil || got != (swapSteps{}) {
		t.Errorf("plan = %+v, %v, want none", got, err)
	}
}
