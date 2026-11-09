package bar

import (
	"testing"
	"testing/fstest"
)

func TestReadMemory(t *testing.T) {
	data := "MemTotal:       1000 kB\nMemFree:  10 kB\nMemAvailable:    250 kB\n"
	got, err := readMemory(fstest.MapFS{"proc/meminfo": {Data: []byte(data)}})
	if err != nil || got != 75 {
		t.Errorf("readMemory = %d, %v; want 75", got, err)
	}
	if _, err := readMemory(fstest.MapFS{"proc/meminfo": {Data: []byte("MemFree: 10 kB\n")}}); err == nil {
		t.Error("readMemory took a meminfo with no MemTotal")
	}
}
