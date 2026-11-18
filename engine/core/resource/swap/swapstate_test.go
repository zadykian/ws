package swap

import "testing"

// file8 is a swap file of 8192 MiB, root's and mode 0600.
var file8 = swapFileInfo{Exists: true, Size: 8192 * mib, Mode: 0o600}

func TestStale(t *testing.T) {
	for name, tt := range map[string]struct {
		s    swapState
		want bool
	}{
		"no file":         {swapState{WantMiB: 1024}, false},
		"right size":      {swapState{WantMiB: 8192, File: file8}, false},
		"another size":    {swapState{WantMiB: 4096, File: file8}, true},
		"size 0, a file":  {swapState{File: file8}, true},
		"size 0, no file": {swapState{}, false},
	} {
		if got := tt.s.stale(); got != tt.want {
			t.Errorf("%s: stale = %v, want %v", name, got, tt.want)
		}
	}
}
