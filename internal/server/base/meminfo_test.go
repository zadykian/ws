package base

import "testing"

func TestMeminfo(t *testing.T) {
	const text = "MemTotal:       16314752 kB\nMemFree:          200 kB\nMemAvailable:    8000000 kB\nHugePages_Total:       0\n"
	for field, want := range map[string]int64{"MemTotal": 16314752, "MemAvailable": 8000000, "HugePages_Total": 0} {
		if got, err := meminfo(text, field); err != nil || got != want {
			t.Errorf("%s = %d, %v, want %d", field, got, err, want)
		}
	}
	for _, field := range []string{"Mem", "SwapTotal"} {
		if _, err := meminfo(text, field); err == nil {
			t.Errorf("%s: no error for a missing field", field)
		}
	}
	if _, err := meminfo("MemTotal: lots kB\n", "MemTotal"); err == nil {
		t.Error("no error for a value that is no number")
	}
}

func TestRAMSwapMiB(t *testing.T) {
	for kib, want := range map[int64]int64{
		0: 0, 1: 0, 1024: 1024, 1048576: 1024, 1048576 + 1024: 2048, 16314752: 16384, 16777216: 16384, 16777216 + 1024: 17408,
	} {
		if got := ramSwapMiB(kib); got != want {
			t.Errorf("ramSwapMiB(%d) = %d, want %d", kib, got, want)
		}
	}
}
