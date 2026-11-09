package bar

import (
	"testing"
	"testing/fstest"
	"time"
)

func TestLineShowsEachSourceThatGivesAReading(t *testing.T) {
	fsys := sensors()
	fsys["proc/stat"] = &fstest.MapFile{Data: []byte("cpu 100 0 100 700 100 0 0 0\n")}
	fsys["proc/meminfo"] = &fstest.MapFile{Data: []byte("MemTotal: 1000 kB\nMemAvailable: 250 kB\n")}
	b := &Bar{FS: fsys, Interval: 10 * time.Second}
	b.gpus.now = time.Now
	if got, want := b.Line(), "cpu 52C  ram 75% 38C  nvme 44C"; got != want {
		t.Errorf("first line = %q, want %q", got, want)
	}
	fsys["proc/stat"] = &fstest.MapFile{Data: []byte("cpu 130 0 120 740 110 0 0 0\n")}
	fsys["run/reboot-required"] = &fstest.MapFile{}
	b.gpus.record("0, 3, 40")
	if got, want := b.Line(), "cpu 50% 52C  gpu 3% 40C  ram 75% 38C  nvme 44C  reboot"; got != want {
		t.Errorf("second line = %q, want %q", got, want)
	}
}

func TestLineOfAVM(t *testing.T) {
	b := &Bar{FS: fstest.MapFS{"proc/meminfo": {Data: []byte("MemTotal: 1000 kB\nMemAvailable: 900 kB\n")}}}
	b.gpus.now = time.Now
	if got, want := b.Line(), "ram 10%"; got != want {
		t.Errorf("line = %q, want %q", got, want)
	}
}
