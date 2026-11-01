package base

import "testing"

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
	} {
		got, err := planSwap(tt.s)
		if err != nil || got != tt.want {
			t.Errorf("%s: plan = %+v, %v, want %+v", tt.name, got, err, tt.want)
		}
	}
}
